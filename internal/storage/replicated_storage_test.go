package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"sort"
	"testing"
)

type memoryReadCloser struct {
	*bytes.Reader
}

func (memoryReadCloser) Close() error {
	return nil
}

type memoryStorageNode struct {
	objects map[string][]byte
	openErr error
}

func newMemoryStorageNode() *memoryStorageNode {
	return &memoryStorageNode{objects: make(map[string][]byte)}
}

func (m *memoryStorageNode) Save(id string, src io.Reader) (uint64, error) {
	data, err := io.ReadAll(src)
	if err != nil {
		return 0, err
	}
	m.objects[id] = append([]byte(nil), data...)
	return uint64(len(data)), nil
}

func (m *memoryStorageNode) Open(id string) (io.ReadCloser, error) {
	if m.openErr != nil {
		return nil, m.openErr
	}
	data, ok := m.objects[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	return memoryReadCloser{Reader: bytes.NewReader(data)}, nil
}

func (m *memoryStorageNode) List() ([]string, error) {
	objects := make([]string, 0, len(m.objects))
	for id := range m.objects {
		objects = append(objects, id)
	}
	sort.Strings(objects)
	return objects, nil
}

func (m *memoryStorageNode) Delete(id string) error {
	if _, ok := m.objects[id]; !ok {
		return os.ErrNotExist
	}
	delete(m.objects, id)
	return nil
}

func (m *memoryStorageNode) Checksum(id string) (string, error) {
	data, ok := m.objects[id]
	if !ok {
		return "", os.ErrNotExist
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func (m *memoryStorageNode) Exists(id string) (bool, error) {
	_, ok := m.objects[id]
	return ok, nil
}

func TestReplicatedStorageOpenFallsBackToHealthyNode(t *testing.T) {
	first := newMemoryStorageNode()
	first.openErr = errors.New("node unavailable")
	second := newMemoryStorageNode()
	second.objects[testObjectID] = []byte("healthy replica")

	replicated := NewReplicatedStorage(first, second)
	file, err := replicated.Open(testObjectID)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer file.Close()

	got, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(got) != "healthy replica" {
		t.Fatalf("Open() content = %q, want %q", got, "healthy replica")
	}
}

func TestReplicatedStorageSyncRepairsMissingAndCorruptReplicas(t *testing.T) {
	correct := []byte("authoritative object data")
	corrupt := []byte("corrupt data")
	hash := sha256.Sum256(correct)
	expectedChecksum := hex.EncodeToString(hash[:])

	source := newMemoryStorageNode()
	source.objects[testObjectID] = append([]byte(nil), correct...)
	corruptNode := newMemoryStorageNode()
	corruptNode.objects[testObjectID] = append([]byte(nil), corrupt...)
	missingNode := newMemoryStorageNode()

	replicated := NewReplicatedStorage(source, corruptNode, missingNode)
	err := replicated.Sync(map[string]string{testObjectID: expectedChecksum})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	for i, node := range []*memoryStorageNode{source, corruptNode, missingNode} {
		checksum, err := node.Checksum(testObjectID)
		if err != nil {
			t.Fatalf("node %d Checksum() error = %v", i, err)
		}
		if checksum != expectedChecksum {
			t.Fatalf("node %d checksum = %q, want %q", i, checksum, expectedChecksum)
		}
	}
}

func TestReplicatedStorageSyncFailsWithoutValidReplica(t *testing.T) {
	nodeOne := newMemoryStorageNode()
	nodeOne.objects[testObjectID] = []byte("corrupt one")
	nodeTwo := newMemoryStorageNode()
	nodeTwo.objects[testObjectID] = []byte("corrupt two")

	wanted := sha256.Sum256([]byte("expected content"))
	replicated := NewReplicatedStorage(nodeOne, nodeTwo)

	err := replicated.Sync(map[string]string{
		testObjectID: hex.EncodeToString(wanted[:]),
	})
	if err == nil {
		t.Fatal("Sync() error = nil, want no-valid-replica error")
	}
}
