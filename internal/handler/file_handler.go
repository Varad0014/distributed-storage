package handler

import (
	"encoding/json"
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
	name := strings.TrimPrefix(r.URL.Path, "/files/")
	if name == ""{
		http.Error(w, "file name not valid", http.StatusBadRequest)
	}
	switch r.Method{
	case http.MethodGet:
		fh.download(w, r, name)
	case http.MethodDelete:
		fh.delete(w, r, name)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (fh *FileHandler) upload(w http.ResponseWriter, r *http.Request){
	// if memory less than maxMemory, stored in RAM, otherwise temp file
	maxMemory := (32<<20) //32Mb
	if err := r.ParseMultipartForm(int64(maxMemory)); err != nil{
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	
	file, header, err := r.FormFile("file")
	if err != nil{
		http.Error(w, "file not valid", http.StatusBadRequest)
		return
	}

	defer file.Close()
	result, err := fh.fileService.Upload(header.Filename, file)
	if err != nil{
		http.Error(w, "Error uploading file", http.StatusInternalServerError)
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (fh *FileHandler) list(w http.ResponseWriter, r *http.Request){

	filesList, err := fh.fileService.List()
	if err != nil{
		http.Error(w, "Could not retrive list", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(filesList)
}

func (fh *FileHandler) delete(w http.ResponseWriter, r *http.Request, name string){

	filesList, err := fh.fileService.Delete(name)
	if err != nil{
		http.Error(w, "Could not find list", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(filesList)
}

func (fh *FileHandler) download(w http.ResponseWriter, r *http.Request, name string){
	file, err := fh.fileService.Open(name)
	if err != nil{
		http.Error(w, "Could not find for download", http.StatusNotFound)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil{
		http.Error(w, "Could not get fule info", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename=\"" + info.Name() + "\"")
	
	http.ServeContent(
		w,
		r,
		info.Name(),
		info.ModTime(),
		file,
	)

}