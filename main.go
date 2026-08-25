package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"log"
	"errors"

	"github.com/Varad0014/distributed-storage/internal/config"
	"github.com/Varad0014/distributed-storage/internal/handler"
	"github.com/Varad0014/distributed-storage/internal/repository"
	"github.com/Varad0014/distributed-storage/internal/service"
	"github.com/Varad0014/distributed-storage/internal/storage"
	"github.com/jackc/pgx/v5"
)

func main() {
	storageDir := "./storage"
	err := os.MkdirAll(storageDir, 0755)
	if err != nil {
		panic(err)
	}
	cfg := config.LoadConfig()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := pgx.Connect(ctx, cfg.DB_URL)
	if err != nil {
		panic(err)
	}
	defer db.Close(ctx)
	fileRepository := repository.NewFileRepository(db)
	err = fileRepository.CreateTable(ctx)
	if err != nil {
		panic(err)
	}
	// localStorage := storage.NewLocalStorage(storageDir)
	node1 := storage.NewRemoteStorage(cfg.STORAGE_NODE_URL)
	node2 := storage.NewRemoteStorage(cfg.STORAGE_NODE_URL_2)

	replicatedStorage := storage.NewReplicatedStorage(node1, node2)
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

	server := http.Server{Addr: ":8080", Handler: apiMux}
	go func() {
		fmt.Println("Server is running on http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()
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

	// Stop the replication worker.
	stop()

	// Close database connection.
	if err := db.Close(context.Background()); err != nil {
		log.Printf("database close failed: %v", err)
	}

	log.Println("server stopped")
}
