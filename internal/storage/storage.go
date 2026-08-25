package storage

import ( 
	"io"
)

type StorageNode interface{
	Save(id string, srcFile io.Reader)(uint64, error)
	Open(id string)(io.ReadCloser, error)
	List()([]string, error) //return list of ids
	Delete(id string)error
	Checksum(id string)(string, error)
	Exists(id string)(bool, error)
}