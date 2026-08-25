package storage

import (
	"fmt"
	"io"
	"os"
)

type ReplicatedStorage struct {
	nodes []StorageNode
	ids   []string
}

type NodeStatus struct {
	Id      string `json:"id"`
	Healthy bool   `json:"healthy"`
	Err     error  `json:"error"`
}

func NewReplicatedStorage(nodes ...StorageNode) *ReplicatedStorage {
	ids := make([]string, len(nodes))
	for i := range nodes {
		ids[i] = fmt.Sprintf("storage-node-%d", i+1)
	}
	return &ReplicatedStorage{nodes: nodes, ids: ids}
}

func (rs *ReplicatedStorage) Save(id string, srcFile io.Reader) (uint64, error) {
	tempFile, err := os.CreateTemp("", "replicated_storage_*")
	if err != nil {
		return 0, err
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	size, err := io.Copy(tempFile, srcFile)
	if err != nil {
		tempFile.Close()
		return 0, err
	}
	if err := tempFile.Close(); err != nil {
		return 0, err
	}

	savedNodes := make([]StorageNode, 0, len(rs.nodes))

	for _, node := range rs.nodes {
		tempFile, err := os.Open(tempFile.Name())
		if err != nil {
			for _, savedNode := range savedNodes {
				savedNode.Delete(id)
			}
			return 0, err
		}
		_, err = node.Save(id, tempFile)
		tempFile.Close()
		if err != nil {
			for _, savedNode := range savedNodes {
				savedNode.Delete(id)
			}
			return 0, fmt.Errorf("failed to save object to node: %v", err)
		}
		savedNodes = append(savedNodes, node)
	}
	return uint64(size), nil
}

func (rs *ReplicatedStorage) Open(id string) (io.ReadCloser, error) {
	var lastErr error
	for _, node := range rs.nodes {
		file, err := node.Open(id)
		if err == nil {
			return file, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("failed to open object from all nodes: %v", lastErr)
}

func (rs *ReplicatedStorage) Delete(id string) error {
	var firstErr error
	for _, node := range rs.nodes {
		err := node.Delete(id)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (rs *ReplicatedStorage) Checksum(id string) (string, error) {
	var lastErr error
	for _, node := range rs.nodes {
		checksum, err := node.Checksum(id)
		if err == nil {
			return checksum, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("failed to get checksum from all nodes: %v", lastErr)
}

func (rs *ReplicatedStorage) Exists(id string) (bool, error) {
	var lastErr error
	for _, node := range rs.nodes {
		exists, err := node.Exists(id)
		if err == nil {
			return exists, nil
		}
		lastErr = err
	}
	return false, fmt.Errorf("failed to check existence from all nodes: %v", lastErr)
}

func (rs *ReplicatedStorage) NodeStatuses() []NodeStatus {
	statuses := make([]NodeStatus, len(rs.nodes))

	for i, node := range rs.nodes {
		status := NodeStatus{
			Id:      rs.ids[i],
			Healthy: false,
		}

		checker, ok := node.(HealthChecker)
		if ok && checker.HealthCheck() == nil {
			status.Healthy = true
		}
		statuses[i] = status
	}

	return statuses
}

func (rs *ReplicatedStorage) List() ([]string, error) {
	objectSet := make(map[string]struct{})

	for _, node := range rs.nodes {
		objects, err := node.List()
		if err != nil {
			continue
		}

		for _, object := range objects {
			objectSet[object] = struct{}{}
		}
	}

	result := make([]string, 0, len(objectSet))

	for object := range objectSet {
		result = append(result, object)
	}

	return result, nil
}

func (rs *ReplicatedStorage) Sync() error {
	if len(rs.nodes) < 2 {
		return nil
	}

	for i, source := range rs.nodes {
		sourceObjects, err := source.List()
		if err != nil {
			continue
		}

		for j, destination := range rs.nodes {
			if i == j {
				continue
			}

			destinationObjects, err := destination.List()
			if err != nil {
				continue
			}

			destinationSet := make(map[string]struct{})

			for _, id := range destinationObjects {
				destinationSet[id] = struct{}{}
			}

			for _, id := range sourceObjects {
				// File doesn't exist on destination.
				if _, exists := destinationSet[id]; !exists {
					if err := rs.replicateObject(
						source,
						destination,
						id,
					); err != nil {
						return err
					}

					continue
				}

				// File exist on both
				sourceChecksum, err := source.Checksum(id)
				if err != nil {
					continue
				}

				destinationChecksum, err := destination.Checksum(id)
				if err != nil {
					continue
				}

				if sourceChecksum != destinationChecksum {
					// Corrupt
					if err := rs.replicateObject(source, destination, id); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (rs *ReplicatedStorage) replicateObject(source StorageNode, destination StorageNode, id string) error {
	file, err := source.Open(id)
	if err != nil {
		return fmt.Errorf(
			"failed to open %s from source: %w",
			id,
			err,
		)
	}
	defer file.Close()

	_, err = destination.Save(id, file)
	if err != nil {
		return fmt.Errorf(
			"failed to replicate %s: %w",
			id,
			err,
		)
	}

	return nil
}
