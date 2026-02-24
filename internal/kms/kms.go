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

// Encryptor holds an AES-256-GCM master key.
type Encryptor struct {
	key []byte // 32 bytes
}

// New creates an Encryptor from a 64-char hex-encoded 32-byte master key.
func New(hexKey string) (*Encryptor, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("kms: decode key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("kms: master key must be 32 bytes (got %d)", len(key))
	}
	return &Encryptor{key: key}, nil
}

// Encrypt encrypts plaintext with AES-256-GCM and returns hex(nonce || ciphertext).
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
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
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a hex(nonce || ciphertext) blob produced by Encrypt.
func (e *Encryptor) Decrypt(hexCiphertext string) (string, error) {
	data, err := hex.DecodeString(hexCiphertext)
	if err != nil {
		return "", fmt.Errorf("kms: decode hex: %w", err)
	}
	block, err := aes.NewCipher(e.key)
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
