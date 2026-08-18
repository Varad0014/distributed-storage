package service

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"github.com/Varad0014/distributed-storage/internal/model"
	"github.com/Varad0014/distributed-storage/internal/storage"
	"github.com/google/uuid"
)


type FileService struct{
	localStorage *storage.LocalStorage
}

func NewFileService(storage *storage.LocalStorage)(*FileService){
	return &FileService{localStorage: storage}
}


func (fs *FileService) Upload(name string, file io.Reader)(*model.File, error){
	// make sure filename is used
	name = filepath.Base(name)
	if name == "." || name == ""{
		return nil, errors.New("Invalid file name")
	}
	bytes, err := fs.localStorage.Save(name, file)
	if err != nil{
		return nil, err
	}
	return &model.File{
		Id: uuid.NewString(),
		Name: name,
		Size: bytes,
	}, nil
}

func (fs *FileService) List()([]model.File, error){
	files, err := fs.localStorage.List()
	if err != nil{
		return nil, err
	}
	fileList := make([]model.File, 0)
	for _, file := range files{
		fileEntry := model.File{
			Id: "",
			Name: file.Name(),
			Size: uint64(file.Size()),
		}
		fileList = append(fileList, fileEntry)
	}
	return fileList, nil
}

func (fs *FileService) Open(name string)(*os.File, error){
	name = filepath.Base(name)
	if name == "." || name == ""{
		return nil, errors.New("Invalid file name")
	}
	return fs.localStorage.Open(name)
}

func (fs *FileService) Delete(name string) ([]model.File, error){
	name = filepath.Base(name)
	if name == "." || name == ""{
		return nil, errors.New("Invalid file name")
	}
	err := fs.localStorage.Delete(name)
	if err != nil{
		return nil, err
	}
	fileList, err := fs.List()
	if err != nil{
		return nil, err
	}
	return fileList, nil
}