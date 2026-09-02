package storage

import (
	"errors"
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
	if len(rs.nodes) == 0 {
		return nil, fmt.Errorf(
			"no storage nodes configured",
		)
	}

	var nodeErrors []error
	missingCount := 0

	for i, node := range rs.nodes {
		file, err := node.Open(id)
		if err == nil {
			return file, nil
		}

		if errors.Is(err, os.ErrNotExist) {
			missingCount++
		}

		nodeErrors = append(
			nodeErrors,
			fmt.Errorf(
				"node %d: %w",
				i,
				err,
			),
		)
	}

	if missingCount == len(rs.nodes) {
		return nil, fmt.Errorf(
			"object %s does not exist on any node: %w",
			id,
			os.ErrNotExist,
		)
	}

	return nil, fmt.Errorf(
		"object %s unavailable from all nodes: %w",
		id,
		errors.Join(nodeErrors...),
	)
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

func (rs *ReplicatedStorage) Sync(expectedChecksums map[string]string) error {
	var syncErrors []error

	for id, expectedChecksum := range expectedChecksums {
		if err := rs.repairObject(id, expectedChecksum); err != nil {
			syncErrors = append(syncErrors, err)
		}
	}

	return errors.Join(syncErrors...)
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

func (rs *ReplicatedStorage) repairObject(id string, expectedChecksum string) error {
	sourceIndex := -1

	// Find a replica that agrees with PostgreSQL.
	for i, node := range rs.nodes {
		checksum, err := node.Checksum(id)
		if err != nil {
			continue
		}

		if checksum == expectedChecksum {
			sourceIndex = i
			break
		}
	}

	if sourceIndex == -1 {
		return fmt.Errorf(
			"no valid replica found for object %s",
			id,
		)
	}

	source := rs.nodes[sourceIndex]
	var repairErrors []error

	for i, destination := range rs.nodes {
		if i == sourceIndex {
			continue
		}

		checksum, err := destination.Checksum(id)

		// This replica is already correct.
		if err == nil && checksum == expectedChecksum {
			continue
		}

		// Replica is missing, unavailable, or corrupted.
		if err := rs.replicateObject(
			source,
			destination,
			id,
		); err != nil {
			repairErrors = append(
				repairErrors,
				fmt.Errorf("repair node %d: %w", i, err),
			)
			continue
		}

		// Do not assume that Save produced correct bytes.
		repairedChecksum, err := destination.Checksum(id)
		if err != nil {
			repairErrors = append(
				repairErrors,
				fmt.Errorf(
					"verify repaired object on node %d: %w",
					i,
					err,
				),
			)
			continue
		}

		if repairedChecksum != expectedChecksum {
			repairErrors = append(
				repairErrors,
				fmt.Errorf(
					"repair verification failed on node %d",
					i,
				),
			)
		}
	}

	return errors.Join(repairErrors...)
}
