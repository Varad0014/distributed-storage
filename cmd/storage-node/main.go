package main

import (
	"fmt"
	"net/http"

	"github.com/Varad0014/distributed-storage/internal/storage"
)

func main() {
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
	fmt.Println("Storage node server is running on http://localhost:8081")
	if err := http.ListenAndServe(":8081", storageNodeMux); err != nil {
		panic(err)
	}
}
