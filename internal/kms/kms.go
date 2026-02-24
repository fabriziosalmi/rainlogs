// Package kms provides AES-256-GCM envelope encryption for secrets at rest.
// The master key is a 32-byte value stored as a 64-char hex string in the
// KMS_MASTER_KEY environment variable.
// For production, replace with AWS KMS, Google Cloud KMS, or HashiCorp Vault.
package kms

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// Encryptor holds multiple AES-256-GCM master keys for rotation support.
type Encryptor struct {
	keys        map[string][]byte // map[id]key
	activeKeyID string
}

// New creates an Encryptor. keys is a map of ID -> hexKey.
// activeKeyID is the ID to use for new encryptions.
func New(hexKey string) (*Encryptor, error) {
	// Backward compatibility: use "v1" as default ID for single key
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("kms: decode key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("kms: master key must be 32 bytes (got %d)", len(key))
	}

	return &Encryptor{
		keys: map[string][]byte{
			"v1": key,
		},
		activeKeyID: "v1",
	}, nil
}

// NewKeyRing creates an Encryptor with multiple keys.
func NewKeyRing(keys map[string]string, activeID string) (*Encryptor, error) {
	decodedKeys := make(map[string][]byte)
	for id, k := range keys {
		decoded, err := hex.DecodeString(k)
		if err != nil {
			return nil, fmt.Errorf("kms: decode key %s: %w", id, err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("kms: key %s must be 32 bytes", id)
		}
		decodedKeys[id] = decoded
	}

	if _, ok := decodedKeys[activeID]; !ok {
		return nil, fmt.Errorf("kms: active key %s not found in keyring", activeID)
	}

	return &Encryptor{
		keys:        decodedKeys,
		activeKeyID: activeID,
	}, nil
}

// Encrypt encrypts plaintext with the active key and returns "id:hex(nonce || ciphertext)".
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	key := e.keys[e.activeKeyID]
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("kms: aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("kms: gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("kms: nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return fmt.Sprintf("%s:%s", e.activeKeyID, hex.EncodeToString(ciphertext)), nil
}

// Decrypt decrypts a string.
// Supported formats:
// 1. "id:hex_ciphertext" (Key rotation format)
// 2. "hex_ciphertext" (Legacy v1 format)
func (e *Encryptor) Decrypt(input string) (string, error) {
	var keyID string
	var hexCiphertext string

	// Check if input has key ID prefix
	// Simple check: does it contain a colon? hex encoded strings don't have colons.
	// But we should be careful about short strings.
	// Assuming ID is alphanumeric.

	// Fast path: check for colon
	var hasColon bool
	for i := 0; i < len(input); i++ {
		if input[i] == ':' {
			hasColon = true
			keyID = input[:i]
			hexCiphertext = input[i+1:]
			break
		}
	}

	// Fallback to v1 (legacy) if no colon found
	if !hasColon {
		keyID = "v1"
		hexCiphertext = input
	}

	key, ok := e.keys[keyID]
	if !ok {
		return "", fmt.Errorf("kms: key id %s not found", keyID)
	}

	data, err := hex.DecodeString(hexCiphertext)
	if err != nil {
		return "", fmt.Errorf("kms: decode hex: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("kms: aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("kms: gcm: %w", err)
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return "", fmt.Errorf("kms: ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, data[:ns], data[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("kms: decrypt: %w", err)
	}
	return string(plaintext), nil
}
