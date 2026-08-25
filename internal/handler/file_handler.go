package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"errors"
	"io"
	"github.com/Varad0014/distributed-storage/internal/service"
	"github.com/Varad0014/distributed-storage/internal/config"
	appErrors "github.com/Varad0014/distributed-storage/internal/errors"
)


type FileHandler struct{
	fileService *service.FileService
	cfg *config.Config
}

func NewFileHandler(fileService *service.FileService, config *config.Config) *FileHandler{
	return &FileHandler{fileService: fileService, cfg: config}
}

func (fh *FileHandler) Files(w http.ResponseWriter, r *http.Request){
	switch r.Method{
	case http.MethodGet:
		fh.list(w, r)
	case http.MethodPost:
		fh.upload(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
func (fh *FileHandler) File(w http.ResponseWriter, r *http.Request){
	id := strings.TrimPrefix(r.URL.Path, "/files/")
	if id == ""{
		http.Error(w, "file ID not valid", http.StatusBadRequest)
		return
	}
	switch r.Method{
	case http.MethodGet:
		fh.download(w, r, id)
	case http.MethodDelete:
		fh.delete(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (fh *FileHandler) upload(w http.ResponseWriter, r *http.Request){
	// if memory less than maxMemory, stored in RAM, otherwise temp file
	maxMemory := (32<<20) //32Mb
	if err := r.ParseMultipartForm(int64(maxMemory)); err != nil{
		http.Error(w, "Invalid form", http.StatusBadRequest)
		fmt.Println(err)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil{
		http.Error(w, "file not valid", http.StatusBadRequest)
		fmt.Println(err)
		return
	}
	if uint64(header.Size) > fh.cfg.MAX_UPLOAD_SIZE_BYTES{
		http.Error(w, "file size exceeds limit", http.StatusBadRequest)
		fmt.Println("file size exceeds limit")
		return
	}
	defer file.Close()
	result, err := fh.fileService.Upload(r.Context(), header.Filename, file)
	if err != nil{
		switch {
		case errors.Is(err, appErrors.ErrFileTooLarge):
			http.Error(w, "file size exceeds limit", http.StatusRequestEntityTooLarge)
		case errors.Is(err, appErrors.ErrInvalidFileName):
			http.Error(w, "invalid file name", http.StatusBadRequest)
		default:
			fmt.Println(err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		fmt.Println(err)
		return
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (fh *FileHandler) list(w http.ResponseWriter, r *http.Request){

	filesList, err := fh.fileService.List(r.Context())
	if err != nil{
		http.Error(w, "Could not retrive list", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(filesList)
}

func (fh *FileHandler) delete(w http.ResponseWriter, r *http.Request, id string){

	filesList, err := fh.fileService.Delete(r.Context(), id)
	if err != nil{
		if errors.Is(err, appErrors.ErrFileNotFound){
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		fmt.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(filesList)
}

func (fh *FileHandler) download(w http.ResponseWriter, r *http.Request, id string){
	fileMeta, file, err := fh.fileService.Get(r.Context(), id)
	if err != nil{
		if errors.Is(err, appErrors.ErrFileNotFound){
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileMeta.Name))
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := io.Copy(w, file); err != nil{
		http.Error(w, "Failed to send file", http.StatusInternalServerError)
		return
	}

}

func (fh *FileHandler) StorageHealth(w http.ResponseWriter, r *http.Request) {
	statuses := fh.fileService.StorageStatuses()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(statuses); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (fh *FileHandler) SyncStorage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	if err := fh.fileService.SyncStorage(); err != nil {
		http.Error(
			w,
			"storage synchronization failed",
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("storage synchronized\n"))
}