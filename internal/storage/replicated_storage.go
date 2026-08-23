package storage

import (
	"fmt"
	"io"
	"os"
)

type ReplicatedStorage struct{
	nodes []StorageNode
}

func NewReplicatedStorage(nodes ...StorageNode) *ReplicatedStorage{
	return &ReplicatedStorage{nodes: nodes}
}

func (rs *ReplicatedStorage) Save(id string, srcFile io.Reader)(uint64, error){
	tempFile, err := os.CreateTemp("", "replicated_storage_*")
	if err != nil{
		return 0, err
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	size, err := io.Copy(tempFile, srcFile)
	if err != nil{
		tempFile.Close()
		return 0, err
	}
	if err := tempFile.Close(); err != nil{
		return 0, err
	}

	savedNodes := make([]StorageNode, 0, len(rs.nodes))

	for _, node := range rs.nodes{
		tempFile, err := os.Open(tempFile.Name())
		if err != nil{
			for _, savedNode := range savedNodes{
				savedNode.Delete(id)
			}
			return 0, err
		}
		_, err = node.Save(id, tempFile)
		tempFile.Close()
		if err != nil{
			for _, savedNode := range savedNodes{
				savedNode.Delete(id)
			}
			return 0, fmt.Errorf("failed to save object to node: %v", err)
		}
		savedNodes = append(savedNodes, node)
	}
	return uint64(size), nil
}

func (rs *ReplicatedStorage) Open(id string)(io.ReadCloser, error){
	var lastErr error
	for _, node := range rs.nodes{
		file, err := node.Open(id)
		if err == nil{
			return file, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("failed to open object from all nodes: %v", lastErr)
}

func (rs *ReplicatedStorage) Delete(id string)(error){
	var firstErr error
	for _, node := range rs.nodes{
		err := node.Delete(id)
		if err != nil{
			if firstErr == nil{
				firstErr = err
			}
		}
	}
	return firstErr
}

func (rs *ReplicatedStorage) Checksum(id string)(string, error){
	var lastErr error
	for _, node := range rs.nodes{
		checksum, err := node.Checksum(id)
		if err == nil{
			return checksum, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("failed to get checksum from all nodes: %v", lastErr)
}

func (rs *ReplicatedStorage) Exists(id string)(bool, error){
	var lastErr error
	for _, node := range rs.nodes{
		exists, err := node.Exists(id)
		if err == nil{
			return exists, nil
		}
		lastErr = err
	}
	return false, fmt.Errorf("failed to check existence from all nodes: %v", lastErr)
}
