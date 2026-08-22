package storage

import (
	"fmt"
	"net/http"
	"strings"
	"io"
	"errors"
	"os"
)


type StorageNodeServer struct{
	storage StorageNode
}

func NewStorageNodeServer(storage StorageNode) *StorageNodeServer{
	return &StorageNodeServer{storage: storage}
}

func (s *StorageNodeServer) Health(w http.ResponseWriter, r *http.Request){
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Storage node is healthy")
}

func (s *StorageNodeServer) Put(w http.ResponseWriter, r *http.Request){
	id := strings.TrimPrefix(r.URL.Path, "/objects/")
	if id == ""{
		http.Error(w, "object ID not valid", http.StatusBadRequest)
		return
	}
	size, err := s.storage.Save(id, r.Body)
	if err != nil{
		http.Error(w, "Failed to save object", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"id": "%s", "size": %d}`, id, size)
}

func (s *StorageNodeServer) Get(w http.ResponseWriter, r *http.Request){
	id := strings.TrimPrefix(r.URL.Path, "/objects/")
	if id == ""{
		http.Error(w, "object ID not valid", http.StatusBadRequest)
		return
	}
	file, err := s.storage.Open(id)
	if err != nil{
		if errors.Is(err, os.ErrNotExist){
			http.Error(w, "Object not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to open object", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	w.Header().Set("Content-type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, file); err != nil{
		http.Error(w, "Failed to send object", http.StatusInternalServerError)
		return
	}
}

func (s *StorageNodeServer) Delete(w http.ResponseWriter, r *http.Request){
	id := strings.TrimPrefix(r.URL.Path, "/objects/")
	if id == ""{
		http.Error(w, "object ID not valid", http.StatusBadRequest)
		return
	}
	err := s.storage.Delete(id)
	if err != nil{
		if errors.Is(err, os.ErrNotExist){
			http.Error(w, "Object not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete object", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *StorageNodeServer) Head(w http.ResponseWriter, r *http.Request){
	id := strings.TrimPrefix(r.URL.Path, "/objects/")
	if id == ""{
		http.Error(w, "object ID not valid", http.StatusBadRequest)
		return
	}
	exists, err := s.storage.Exists(id)
	if err != nil{
		http.Error(w, "Failed to check object existence", http.StatusInternalServerError)
		return
	}
	if !exists{
		http.Error(w, "Object not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *StorageNodeServer) Checksum(w http.ResponseWriter, r *http.Request){
	id := strings.TrimPrefix(r.URL.Path, "/objects-checksum/")
	if id == ""{
		http.Error(w, "object ID not valid", http.StatusBadRequest)
		return
	}
	checksum, err := s.storage.Checksum(id)
	if err != nil{
		if errors.Is(err, os.ErrNotExist){
			http.Error(w, "Object not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to calculate checksum", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"id": "%s", "checksum": "%s"}`, id, checksum)
}