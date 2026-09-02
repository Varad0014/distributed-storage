package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(path string) *LocalStorage {
	return &LocalStorage{basePath: path}
}

func (ls *LocalStorage) objectPath(id string) (string, error) {
	if err := validateObjectId(id); err != nil {
		return "", err
	}

	return filepath.Join(ls.basePath, id), nil
}

// os.File implements interface io.Reader, io.Writer
func (ls *LocalStorage) Save(id string, srcFile io.Reader) (uint64, error) {
	path, err := ls.objectPath(id)
	if err != nil {
		return 0, err
	}
	dstFile, err := os.Create(path)

	if err != nil {
		return 0, err
	}
	// remember to close
	defer dstFile.Close()

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return 0, err
	}
	return uint64(bytes), nil
}

func (ls *LocalStorage) Checksum(id string) (string, error) {
	file, err := ls.Open(id)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (ls *LocalStorage) Open(id string) (io.ReadCloser, error) {
	path, err := ls.objectPath(id)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (ls *LocalStorage) Delete(id string) error {
	path, err := ls.objectPath(id)
	if err != nil {
		return err
	}
	fmt.Println(path)
	return os.Remove(path)

}

func (ls *LocalStorage) Exists(id string) (bool, error) {
	path, err := ls.objectPath(id)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (ls *LocalStorage) List() ([]string, error) {
	entries, err := os.ReadDir(ls.basePath)
	if err != nil {
		return nil, err
	}

	objects := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		objects = append(objects, entry.Name())
	}

	return objects, nil
}
