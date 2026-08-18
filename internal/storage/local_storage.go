package storage

import(
	"os"
	"io"
	"path/filepath"
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

// func (ls *LocalStorage) 




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

func (ls *LocalStorage) Open(name string) (*os.File, error){
	path := filepath.Join(ls.basePath, name)
	return os.Open(path)
}

func (ls *LocalStorage) Delete(name string)(error){
	path := filepath.Join(ls.basePath, name)
	return os.Remove(path)

}