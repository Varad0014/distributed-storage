package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Varad0014/distributed-storage/internal/config"
	appErrors "github.com/Varad0014/distributed-storage/internal/errors"
	"github.com/Varad0014/distributed-storage/internal/model"
	"github.com/Varad0014/distributed-storage/internal/repository"
	"github.com/Varad0014/distributed-storage/internal/storage"
	"github.com/google/uuid"
)

type FileService struct {
	storage        storage.StorageNode
	fileRepository *repository.FileRepository
	cfg            *config.Config
}

func NewFileService(storage storage.StorageNode, fileRepository *repository.FileRepository, cfg *config.Config) *FileService {
	return &FileService{storage: storage, fileRepository: fileRepository, cfg: cfg}
}

func (fs *FileService) Upload(ctx context.Context, name string, src io.Reader) (*model.File, error) {
	// make sure filename is used
	name = filepath.Base(name)
	if name == "." || name == "" {
		return nil, appErrors.ErrInvalidFileName
	}

	Id := uuid.NewString()
	//check for max size
	limitReader := io.LimitReader(src, int64(fs.cfg.MAX_UPLOAD_SIZE_BYTES)+1)

	bytes, err := fs.storage.Save(Id, limitReader)
	if err != nil {
		return nil, err
	}
	if bytes > fs.cfg.MAX_UPLOAD_SIZE_BYTES {
		_ = fs.storage.Delete(Id)
		return nil, appErrors.ErrFileTooLarge
	}

	checksum, err := fs.storage.Checksum(Id)
	if err != nil {
		_ = fs.storage.Delete(Id)
		fmt.Println(err)
		return nil, err
	}
	file := &model.File{
		Id:        Id,
		Name:      name,
		Size:      bytes,
		Checksum:  checksum,
		CreatedAt: time.Now(),
	}
	err = fs.fileRepository.Create(ctx, file)
	if err != nil {
		_ = fs.storage.Delete(Id)
		fmt.Println(err)
		return nil, err
	}
	return file, nil
}

func (fs *FileService) List(ctx context.Context) ([]model.File, error) {
	return fs.fileRepository.GetAll(ctx)
}

func (fs *FileService) Get(ctx context.Context, id string) (*model.File, io.ReadCloser, error) {
	fileMeta, err := fs.fileRepository.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	file, err := fs.storage.Open(fileMeta.Id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, appErrors.ErrFileNotFound
		}
		fmt.Println(err)
		return nil, nil, err
	}
	return fileMeta, file, nil
}

func (fs *FileService) Delete(ctx context.Context, id string) ([]model.File, error) {
	_, err := fs.fileRepository.GetByID(ctx, id)
	if err != nil {
		fmt.Println("ID not found")
		return nil, err
	}

	err = fs.storage.Delete(id)
	if err != nil {
		fmt.Println("Delete id not found", err)
		return nil, err
	}

	err = fs.fileRepository.Delete(ctx, id)
	if err != nil {
		return nil, err
	}

	fileList, err := fs.List(ctx)
	if err != nil {
		return nil, err
	}
	return fileList, nil
}

func (fs *FileService) StorageStatuses() []storage.NodeStatus {
	replicated, ok := fs.storage.(*storage.ReplicatedStorage)
	if !ok {
		return nil
	}

	return replicated.NodeStatuses()
}

func (fs *FileService) SyncStorage(ctx context.Context) error {
	replicated, ok := fs.storage.(*storage.ReplicatedStorage)
	if !ok {
		return fmt.Errorf("storage does not support replication")
	}
	files, err := fs.fileRepository.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("get authoritative file metadata: %w", err)
	}
	expectedChecksums := make(map[string]string, len(files))

	for _, file := range files {
		expectedChecksums[file.Id] = file.Checksum
	}

	return replicated.Sync(expectedChecksums)
}
