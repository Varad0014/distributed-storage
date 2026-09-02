package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"
)

const testObjectID = "550e8400-e29b-41d4-a716-446655440000"

func TestLocalStorageLifecycle(t *testing.T) {
	store := NewLocalStorage(t.TempDir())
	content := []byte("distributed storage test data")

	size, err := store.Save(testObjectID, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if size != uint64(len(content)) {
		t.Fatalf("Save() size = %d, want %d", size, len(content))
	}

	exists, err := store.Exists(testObjectID)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Fatal("Exists() = false, want true")
	}

	file, err := store.Open(testObjectID)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	got, err := io.ReadAll(file)
	closeErr := file.Close()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if closeErr != nil {
		t.Fatalf("Close() error = %v", closeErr)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("Open() content = %q, want %q", got, content)
	}

	wantHash := sha256.Sum256(content)
	wantChecksum := hex.EncodeToString(wantHash[:])
	checksum, err := store.Checksum(testObjectID)
	if err != nil {
		t.Fatalf("Checksum() error = %v", err)
	}
	if checksum != wantChecksum {
		t.Fatalf("Checksum() = %q, want %q", checksum, wantChecksum)
	}

	objects, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(objects) != 1 || objects[0] != testObjectID {
		t.Fatalf("List() = %v, want [%s]", objects, testObjectID)
	}

	if err := store.Delete(testObjectID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	exists, err = store.Exists(testObjectID)
	if err != nil {
		t.Fatalf("Exists() after delete error = %v", err)
	}
	if exists {
		t.Fatal("Exists() after delete = true, want false")
	}
}

func TestLocalStorageRejectsInvalidObjectID(t *testing.T) {
	store := NewLocalStorage(t.TempDir())

	if _, err := store.Save("../unsafe", bytes.NewReader([]byte("data"))); err == nil {
		t.Fatal("Save() with invalid object ID returned nil error")
	}
}
