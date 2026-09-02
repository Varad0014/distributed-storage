package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Varad0014/distributed-storage/internal/config"
	"github.com/Varad0014/distributed-storage/internal/storage"
)

func main() {
	cfg := config.LoadConfig()
	if cfg.STORAGE_NODE_TOKEN == "" {
		log.Fatal("STORAGE_NODE_TOKEN is required")
	}
	localStorage := storage.NewLocalStorage("/app/storage")
	storageNodeServer := storage.NewStorageNodeServer(localStorage)
	storageNodeMux := http.NewServeMux()

	storageNodeMux.HandleFunc("/health", storageNodeServer.Health)
	storageNodeMux.HandleFunc("/objects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		storageNodeServer.List(w, r)
	})
	storageNodeMux.HandleFunc("/objects/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			storageNodeServer.Put(w, r)
		case http.MethodGet:
			storageNodeServer.Get(w, r)
		case http.MethodDelete:
			storageNodeServer.Delete(w, r)
		case http.MethodHead:
			storageNodeServer.Head(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	storageNodeMux.HandleFunc("/objects-checksum/{id}", storageNodeServer.Checksum)

	protectedHandler := storage.RequireStorageToken(cfg.STORAGE_NODE_TOKEN, storageNodeMux)
	fmt.Println("Storage node server is running on http://localhost:8081")
	server := http.Server{
		Addr:              ":8081",
		Handler:           protectedHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("storage node server failed: %v", err)
	}
}
