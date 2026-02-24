package worm_test

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fabriziosalmi/rainlogs/pkg/worm"
)

func TestGenesisHash(t *testing.T) {
	assert.Len(t, worm.GenesisHash, 64, "genesis hash must be 64 hex chars")
	assert.Equal(t, strings.Repeat("0", 64), worm.GenesisHash)
}

func TestChainHash_Deterministic(t *testing.T) {
	objHash := "abc123def456abc123def456abc123def456abc123def456abc123def456abc1"
	jobID := "550e8400-e29b-41d4-a716-446655440000"

	h1 := worm.ChainHash(worm.GenesisHash, objHash, jobID)
	h2 := worm.ChainHash(worm.GenesisHash, objHash, jobID)

	assert.Equal(t, h1, h2, "ChainHash must be deterministic")
	assert.Len(t, h1, 64, "ChainHash must return 64 hex chars")
}

func TestChainHash_UniquePerInput(t *testing.T) {
	obj := "abc123def456abc123def456abc123def456abc123def456abc123def456abc1"
	jobA := "550e8400-e29b-41d4-a716-446655440000"
	jobB := "660e8400-e29b-41d4-a716-446655440001"

	hA := worm.ChainHash(worm.GenesisHash, obj, jobA)
	hB := worm.ChainHash(worm.GenesisHash, obj, jobB)

	assert.NotEqual(t, hA, hB, "different job IDs must produce different chain hashes")
}

func TestChainHash_ChainProgression(t *testing.T) {
	obj1 := "1111111111111111111111111111111111111111111111111111111111111111"
	obj2 := "2222222222222222222222222222222222222222222222222222222222222222"
	obj3 := "3333333333333333333333333333333333333333333333333333333333333333"

	h1 := worm.ChainHash(worm.GenesisHash, obj1, "job-1")
	h2 := worm.ChainHash(h1, obj2, "job-2")
	h3 := worm.ChainHash(h2, obj3, "job-3")

	assert.NotEqual(t, h1, h2)
	assert.NotEqual(t, h2, h3)
	assert.NotEqual(t, h1, h3)

	// Tampering mid-chain must invalidate all downstream hashes.
	h2Tampered := worm.ChainHash(h1, strings.Repeat("9", 64), "job-2")
	h3Tampered := worm.ChainHash(h2Tampered, obj3, "job-3")
	assert.NotEqual(t, h2, h2Tampered, "tampered hash must differ from original")
	assert.NotEqual(t, h3, h3Tampered, "downstream hashes must be invalidated after tampering")
}

func TestVerifyObject_Valid(t *testing.T) {
	data := []byte("NIS2 compliance log entry 2024")
	sum := sha256.Sum256(data)
	hexHash := hex.EncodeToString(sum[:])

	err := worm.VerifyObject(data, hexHash)
	require.NoError(t, err)
}

func TestVerifyObject_Mismatch(t *testing.T) {
	data := []byte("some log data")
	err := worm.VerifyObject(data, strings.Repeat("0", 64))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sha256 mismatch")
}

func TestVerifyObject_EmptyData(t *testing.T) {
	// SHA-256 of empty byte slice – well-known value.
	emptyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	err := worm.VerifyObject([]byte{}, emptyHash)
	require.NoError(t, err)
}

func TestVerifyObject_LargeData(t *testing.T) {
	data := []byte(strings.Repeat("x", 1<<20)) // 1 MiB
	sum := sha256.Sum256(data)
	hexHash := hex.EncodeToString(sum[:])

	err := worm.VerifyObject(data, hexHash)
	require.NoError(t, err)
}
