package storage

import(
	"os"
	"io"
	"path/filepath"
	"crypto/sha256"
	"encoding/hex"
)


type LocalStorage struct{
	basePath string
}

func NewLocalStorage(path string) (*LocalStorage){
	return &LocalStorage{basePath: path}
}

// os.File implements interface io.Reader, io.Writer
func (ls *LocalStorage) Save(name string, srcFile io.Reader)(uint64, error){
	path := filepath.Join(ls.basePath, name)
	dstFile, err := os.Create(path)
	if err != nil{
		return 0, err
	}
	// remember to close
	defer dstFile.Close()

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil{
		return 0, err
	}
	return uint64(bytes), nil
}

func (ls *LocalStorage) CheckSum(name string)(string, error){
	path := filepath.Join(ls.basePath, name)
	file, err := os.Open(path)
	if err != nil{
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil{
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}




// return FileInfo, not model.File, we can handle it in service and return File
func (ls *LocalStorage) List()([]os.FileInfo, error){
	entries, err := os.ReadDir(ls.basePath)
	if err != nil{
		return nil, err
	}
	fileList := make([]os.FileInfo, 0)
	for _, entry := range entries{
		if entry.IsDir(){
			continue
		}
		fileInfo, err := entry.Info()
		if err != nil{
			return nil, err
		}
		fileList = append(fileList, fileInfo)
	}
	return fileList, nil
}

func (ls *LocalStorage) Open(id string) (*os.File, error){
	path := filepath.Join(ls.basePath, id)
	return os.Open(path)
}

func (ls *LocalStorage) Delete(id string)(error){
	path := filepath.Join(ls.basePath, id)
	return os.Remove(path)

}