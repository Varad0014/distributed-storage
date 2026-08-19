package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Varad0014/distributed-storage/internal/service"
)


type FileHandler struct{
	fileService *service.FileService
}

func NewFileHandler(fileService *service.FileService) *FileHandler{
	return &FileHandler{fileService: fileService}
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

	defer file.Close()
	result, err := fh.fileService.Upload(r.Context(), header.Filename, file)
	if err != nil{
		http.Error(w, "Error uploading file", http.StatusInternalServerError)
		fmt.Println(err)
		return
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (fh *FileHandler) list(w http.ResponseWriter, r *http.Request){

	filesList, err := fh.fileService.List(r.Context())
	fmt.Println(filesList)
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
		http.Error(w, "Could not find list", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(filesList)
}

func (fh *FileHandler) download(w http.ResponseWriter, r *http.Request, id string){
	fileMeta, file, err := fh.fileService.Get(r.Context(), id)
	if err != nil{
		http.Error(w, "Could not find for download", http.StatusNotFound)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Disposition", "attachment; filename=\"" + fileMeta.Name + "\"")
	
	http.ServeContent(
		w,
		r,
		fileMeta.Name,
		fileMeta.CreatedAt,
		file,
	)

}