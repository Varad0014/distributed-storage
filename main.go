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
	fileRespository := repository.NewFileRepository(db)
	err = fileRespository.CreateTable(ctx)
	if err != nil{
		panic(err)
	}
	localStorage := storage.NewLocalStorage(storageDir)
	fileRepository := repository.NewFileRepository(db)
	fileService := service.NewFileService(localStorage, fileRepository, cfg)
	fileHandler := handler.NewFileHandler(fileService, cfg)
	http.HandleFunc("/files", fileHandler.Files)
	http.HandleFunc("/files/", fileHandler.File)
	fmt.Println("Server is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil{
		panic(err)
	}

}