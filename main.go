package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Varad0014/distributed-storage/internal/config"
	"github.com/Varad0014/distributed-storage/internal/handler"
	"github.com/Varad0014/distributed-storage/internal/repository"
	"github.com/Varad0014/distributed-storage/internal/service"
	"github.com/Varad0014/distributed-storage/internal/storage"
	"github.com/jackc/pgx/v5"
)

func main() {

	// Create storage directory in each node
	storageDir := "./storage"
	err := os.MkdirAll(storageDir, 0755)
	if err != nil {
		// non recoverable
		log.Fatalf("Could not create directory %s: %v", storageDir, err)
	}

	// Load environment variables like postgres url, storage node urls, maximum size allowed
	cfg := config.LoadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := pgx.Connect(ctx, cfg.DB_URL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// defer ignores error status, but to log error, we can use this
	// defer db.Close(ctx)
	defer func() {
		if err := db.Close(ctx); err != nil {
			// Close database connection.
			log.Printf("database close failed: %v", err)
		}
	}()

	fileRepository := repository.NewFileRepository(db)
	err = fileRepository.CreateTable(ctx)
	if err != nil {
		panic(err)
	}
	// localStorage := storage.NewLocalStorage(storageDir)
	nodeURLs := strings.Split(cfg.STORAGE_NODE_URLS, ",")

	nodes := make([]storage.StorageNode, 0, len(nodeURLs))

	for _, url := range nodeURLs {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		nodes = append(
			nodes,
			storage.NewRemoteStorage(url, cfg.STORAGE_NODE_TOKEN),
		)
	}

	if len(nodes) == 0 {
		log.Fatal("No valid storage nodes configured in STORAGE_NODE_URLS")
	}

	replicatedStorage := storage.NewReplicatedStorage(nodes...)
	// remoteStorage := storage.NewRemoteStorage(cfg.STORAGE_NODE_URL)
	fileService := service.NewFileService(replicatedStorage, fileRepository, cfg)
	fileHandler := handler.NewFileHandler(fileService, cfg)
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/files", fileHandler.Files)
	apiMux.HandleFunc("/files/", fileHandler.File)
	apiMux.HandleFunc("/health/storage", fileHandler.StorageHealth)
	apiMux.HandleFunc("/storage/sync", fileHandler.SyncStorage)

	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()
	worker := service.NewReplicationWorker(fileService.SyncStorage, 10*time.Second)
	go worker.Start(workerCtx)

	server := http.Server{
		Addr:              ":8080",
		Handler:           apiMux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		fmt.Println("Server is running on http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// block until Ctrl+C signal
	<-ctx.Done()
	log.Println("shutdown signal received")

	// Stop accepting requests and allow existing requests to finish.
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown failed: %v", err)
	}

	log.Println("server stopped")
}
