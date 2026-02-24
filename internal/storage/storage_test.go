package storage

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFSStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rainlogs-test-storage")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewFSStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	customerID := uuid.New()
	zoneID := uuid.New()
	now := time.Now()
	rawLogs := []byte("{\"event\":\"test1\"}\n{\"event\":\"test2\"}\n")

	// Test PutLogs
	key, sha256hex, size, lines, err := store.PutLogs(ctx, customerID, zoneID, now, now.Add(time.Second), rawLogs)
	if err != nil {
		t.Fatalf("PutLogs failed: %v", err)
	}

	if key == "" {
		t.Error("expected non-empty key")
	}
	if sha256hex == "" {
		t.Error("expected non-empty hash")
	}
	if size == 0 {
		t.Error("expected non-zero size")
	}
	if lines != 2 {
		t.Errorf("expected 2 lines, got %d", lines)
	}

	// Test GetLogs
	readBack, err := store.GetLogs(ctx, key)
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}

	if !bytes.Equal(readBack, rawLogs) {
		t.Errorf("expected %q, got %q", rawLogs, readBack)
	}

	// Test DeleteObject
	if err := store.DeleteObject(ctx, key); err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}

	// Test GetLogs after delete (should fail)
	_, err = store.GetLogs(ctx, key)
	if err == nil {
		t.Error("expected error getting deleted object")
	}
}

func TestPrepareBlob(t *testing.T) {
	raw := []byte("foo\nbar\n")
	cid := uuid.New()
	zid := uuid.New()
	start := time.Now()
	end := start.Add(time.Minute)

	compressed, meta, err := PrepareBlob(raw, cid, zid, start, end)
	if err != nil {
		t.Fatal(err)
	}

	if meta.Lines != 2 {
		t.Errorf("expected 2 lines, got %d", meta.Lines)
	}
	if len(compressed) == 0 {
		t.Error("expected compressed data")
	}

	decompressed, err := DecompressBlob(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decompressed, raw) {
		t.Errorf("roundtrip mismatch: want %q, got %q", raw, decompressed)
	}
}
