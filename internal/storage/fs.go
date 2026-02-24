package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// FSStore implements the Backend interface using the local filesystem.
type FSStore struct {
	root     string
	provider string
}

// NewFSStore creates a new filesystem-based storage backend.
func NewFSStore(root string) (*FSStore, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("storage: invalid root path: %w", err)
	}

	if err := os.MkdirAll(absRoot, 0o755); err != nil {
		return nil, fmt.Errorf("storage: create root dir: %w", err)
	}

	return &FSStore{
		root:     absRoot,
		provider: "filesystem",
	}, nil
}

func (s *FSStore) Provider() string {
	return s.provider
}

func (s *FSStore) PutLogs(_ context.Context, customerID, zoneID uuid.UUID, from, to time.Time, raw []byte, logType string) (key, sha256hex string, compressedBytes, logLines int64, err error) {
	// Re-use logic for compression/hashing/key generation from common helpers?
	// For now, let's duplicate the non-AWS logic to keep it independent,
	// or ideally refactor S3 logic to share "blob preparation".

	// But wait, the interface defines PutLogs taking raw bytes.
	// We should probably share the compression/hashing logic.
	// Let's create `internal/storage/common.go` for shared helpers.
	// For now, let's implement minimal logic here.

	// Actually, let's call the helper to compress and hash.
	// We'll define `PrepareBlob` in `common.go` next.
	blob, meta, err := PrepareBlob(raw, customerID, zoneID, from, to, logType)
	if err != nil {
		return "", "", 0, 0, err
	}

	fullPath := filepath.Join(s.root, meta.Key)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", 0, 0, fmt.Errorf("storage: mkdir: %w", err)
	}

	// Atomic write: write to temp file then rename (same partition)
	tmpFile, err := os.CreateTemp(dir, "rainlog-*.tmp")
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("storage: create temp: %w", err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName) // Cleanup (ignored if renamed successfully)

	if err := tmpFile.Chmod(0o644); err != nil {
		tmpFile.Close()
		return "", "", 0, 0, fmt.Errorf("storage: chmod: %w", err)
	}

	if _, err := tmpFile.Write(blob); err != nil {
		tmpFile.Close()
		return "", "", 0, 0, fmt.Errorf("storage: write temp: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", "", 0, 0, fmt.Errorf("storage: close temp: %w", err)
	}

	if err := os.Rename(tmpName, fullPath); err != nil {
		return "", "", 0, 0, fmt.Errorf("storage: rename: %w", err)
	}

	return meta.Key, meta.SHA256, meta.Size, meta.Lines, nil
}

func (s *FSStore) GetLogs(_ context.Context, key string) ([]byte, error) {
	fullPath := filepath.Join(s.root, key)
	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("storage: key not found: %s", key)
		}
		return nil, fmt.Errorf("storage: open file: %w", err)
	}
	defer f.Close()

	// Need to decompress? Interface implies we return compressed bytes?
	// S3 `GetLogs` implementation returns decompressed bytes:
	// `return io.ReadAll(gr)` where gr is gzip reader.
	// So we should do the same.

	return DecompressBlob(f)
}

func (s *FSStore) DeleteObject(_ context.Context, key string) error {
	fullPath := filepath.Join(s.root, key)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // idempotent delete
		}
		return fmt.Errorf("storage: delete file: %w", err)
	}
	return nil
}
