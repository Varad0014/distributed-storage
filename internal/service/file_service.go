package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Varad0014/distributed-storage/internal/model"
	"github.com/Varad0014/distributed-storage/internal/repository"
	"github.com/Varad0014/distributed-storage/internal/storage"
	"github.com/google/uuid"
)


type FileService struct{
	localStorage *storage.LocalStorage
	fileRepository *repository.FileRepository
}

func NewFileService(localStorage *storage.LocalStorage, fileRepository *repository.FileRepository)(*FileService){
	return &FileService{localStorage: localStorage, fileRepository: fileRepository}
}


func (fs *FileService) Upload(ctx context.Context, name string, src io.Reader)(*model.File, error){
	// make sure filename is used
	name = filepath.Base(name)
	if name == "." || name == ""{
		return nil, errors.New("Invalid file name")
	}
	Id := uuid.NewString()
	bytes, err := fs.localStorage.Save(Id, src)
	if err != nil{
		return nil, err
	}
	checksum, err := fs.localStorage.CheckSum(Id)
	if err != nil{
		_ = fs.localStorage.Delete(Id)
		fmt.Println(err)
		return nil, err
	}
	file := &model.File{
		Id: Id,
		Name: name,
		Size: bytes,
		Checksum: checksum,
		CreatedAt: time.Now(),
	}
	err = fs.fileRepository.Create(ctx, file)
	if err != nil{
		_ = fs.localStorage.Delete(Id)
		fmt.Println(err)
		return nil, err
	}
	return file, nil
}

func (fs *FileService) List(ctx context.Context)([]model.File, error){
	return fs.fileRepository.GetAll(ctx)
}

func (fs *FileService) Get(ctx context.Context, id string)(*model.File, *os.File, error){
	fileMeta, err := fs.fileRepository.GetByID(ctx, id)
	if err != nil{
		return nil, nil, err
	}
	file, err := fs.localStorage.Open(fileMeta.Id)
	if err != nil{
		return nil, nil, err
	}
	return fileMeta, file, nil
}

func (fs *FileService) Delete(ctx context.Context, id string) ([]model.File, error){
	_, err := fs.fileRepository.GetByID(ctx, id)
	if err != nil{
		return nil, err
	}

	err = fs.localStorage.Delete(id)
	if err != nil{
		return nil, err
	}

	err = fs.fileRepository.Delete(ctx, id)
	if err != nil{
		return nil, err
	}
	
	fileList, err := fs.List(ctx)
	if err != nil{
		return nil, err
	}
	return fileList, nil
}