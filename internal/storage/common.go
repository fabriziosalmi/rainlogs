package storage

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

type BlobMetadata struct {
	Key    string
	SHA256 string
	Size   int64
	Lines  int64
}

// PrepareBlob compresses, hashes, and generates a key for raw log data.
func PrepareBlob(raw []byte, customerID, zoneID uuid.UUID, from, to time.Time) ([]byte, BlobMetadata, error) {
	lines := int64(countLines(raw))

	// Compress
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(raw); err != nil {
		return nil, BlobMetadata{}, fmt.Errorf("storage: gzip write: %w", err)
	}
	if err := gw.Close(); err != nil {
		return nil, BlobMetadata{}, fmt.Errorf("storage: gzip close: %w", err)
	}

	compressed := buf.Bytes()
	size := int64(len(compressed))

	// Hash
	sum := sha256.Sum256(compressed)
	sha256hex := hex.EncodeToString(sum[:])

	// Key: logs/<customer>/<zone>/<YYYY>/<MM>/<DD>/<from>_<to>_<sha[:8]>.ndjson.gz
	key := fmt.Sprintf("logs/%s/%s/%s/%s_%s_%s.ndjson.gz",
		customerID,
		zoneID,
		from.UTC().Format("2006/01/02"),
		from.UTC().Format("20060102T150405Z"),
		to.UTC().Format("20060102T150405Z"),
		sha256hex[:8],
	)

	return compressed, BlobMetadata{
		Key:    key,
		SHA256: sha256hex,
		Size:   size,
		Lines:  lines,
	}, nil
}

// DecompressBlob reads gzip compressed data from a reader.
func DecompressBlob(r io.Reader) ([]byte, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("storage: gzip reader: %w", err)
	}
	defer gr.Close()

	return io.ReadAll(gr)
}

func countLines(b []byte) int {
	count := 0
	for _, x := range b {
		if x == '\n' {
			count++
		}
	}
	return count
}
