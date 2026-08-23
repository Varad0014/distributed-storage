package main

import (
	"fmt"
	"net/http"
	"os"
	"context"
	"github.com/Varad0014/distributed-storage/internal/handler"
	"github.com/Varad0014/distributed-storage/internal/repository"
	"github.com/Varad0014/distributed-storage/internal/service"
	"github.com/Varad0014/distributed-storage/internal/storage"
	"github.com/Varad0014/distributed-storage/internal/config"
	"github.com/jackc/pgx/v5"
)

func main(){
	storageDir := "./storage"
	err := os.MkdirAll(storageDir, 0755)
	if err != nil{
		panic(err)
	}
	cfg := config.LoadConfig()
	ctx := context.Background()
	db, err := pgx.Connect(ctx, cfg.DB_URL)
	if err != nil{
		panic(err)
	}
	defer db.Close(ctx)
	fileRepository := repository.NewFileRepository(db)
	err = fileRepository.CreateTable(ctx)
	if err != nil{
		panic(err)
	}
	// localStorage := storage.NewLocalStorage(storageDir)
	node1 := storage.NewRemoteStorage(cfg.STORAGE_NODE_URL)
	node2 := storage.NewRemoteStorage(cfg.STORAGE_NODE_URL_2)

	replicationStorage := storage.NewReplicatedStorage(node1, node2)
	// remoteStorage := storage.NewRemoteStorage(cfg.STORAGE_NODE_URL)
	fileService := service.NewFileService(replicationStorage, fileRepository, cfg)
	fileHandler := handler.NewFileHandler(fileService, cfg)
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/files", fileHandler.Files)
	apiMux.HandleFunc("/files/", fileHandler.File)

	fmt.Println("Server is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", apiMux); err != nil{
		panic(err)
	}



}