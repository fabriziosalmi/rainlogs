// Package worm provides WORM-integrity helpers: a tamper-evident hash chain
// over LogJob records and per-object SHA-256 verification.
package worm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenesisHash is the well-known seed for the first job in a chain.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// ChainHash computes the next link in the audit chain:
// SHA-256(prevChainHash || objectSHA256 || jobID).
func ChainHash(prevChainHash, objectSHA256, jobID string) string {
	h := sha256.New()
	h.Write([]byte(prevChainHash))
	h.Write([]byte(objectSHA256))
	h.Write([]byte(jobID))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyObject confirms that the SHA-256 of data matches expected.
func VerifyObject(data []byte, expectedHex string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != expectedHex {
		return fmt.Errorf("worm: sha256 mismatch: got %s, expected %s", got, expectedHex)
	}
	return nil
}
