package main

import (
	"net/http"
	"os"

	"github.com/Varad0014/distributed-storage/internal/handler"
	"github.com/Varad0014/distributed-storage/internal/service"
	"github.com/Varad0014/distributed-storage/internal/storage"
)

func main(){
	storageDir := "./storage"
	err := os.MkdirAll(storageDir, 0755)
	if err != nil{
		panic(err)
	}
	localStorage := storage.NewLocalStorage(storageDir)
	fileService := service.NewFileService(localStorage)
	fileHandler := handler.NewFileHandler(fileService)
	http.HandleFunc("/files", fileHandler.Files)
	http.HandleFunc("/files/", fileHandler.File)
	if err := http.ListenAndServe(":8080", nil); err != nil{
		panic(err)
	}

}