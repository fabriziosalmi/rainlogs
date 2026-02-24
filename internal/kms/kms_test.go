package kms_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fabriziosalmi/rainlogs/internal/kms"
)

const testKey = "0000000000000000000000000000000000000000000000000000000000000000"

func newTestEncryptor(t *testing.T) *kms.Encryptor {
	t.Helper()
	enc, err := kms.New(testKey)
	require.NoError(t, err)
	return enc
}

func TestNew_ValidKey(t *testing.T) {
	_, err := kms.New(testKey)
	require.NoError(t, err)
}

func TestNew_InvalidHex(t *testing.T) {
	_, err := kms.New("not-valid-hex")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode key")
}

func TestNew_WrongLength(t *testing.T) {
	// 16 bytes = 32 hex chars – too short for AES-256.
	_, err := kms.New(strings.Repeat("0", 32))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	enc := newTestEncryptor(t)
	plaintext := "cf-api-key-ABCDEF123456"

	ciphertext, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEqual(t, plaintext, ciphertext)

	recovered, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, recovered)
}

func TestEncrypt_Nondeterministic(t *testing.T) {
	// AES-GCM uses a random nonce, so two encryptions of the same plaintext
	// must produce different ciphertexts.
	enc := newTestEncryptor(t)
	plaintext := "same-secret"

	c1, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	c2, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	assert.NotEqual(t, c1, c2, "GCM encryption must be non-deterministic (random nonce)")
}

func TestDecrypt_CorruptedCiphertext(t *testing.T) {
	enc := newTestEncryptor(t)

	_, err := enc.Decrypt("not-valid-hex!!!")
	require.Error(t, err)
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	enc := newTestEncryptor(t)
	ct, err := enc.Encrypt("sensitive-value")
	require.NoError(t, err)

	// Flip one byte in the middle of the ciphertext hex.
	tampered := ct[:10] + "ff" + ct[12:]

	_, err = enc.Decrypt(tampered)
	require.Error(t, err, "tampered ciphertext must fail decryption")
}

func TestDecrypt_EmptyInput(t *testing.T) {
	enc := newTestEncryptor(t)
	_, err := enc.Decrypt("")
	require.Error(t, err)
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	enc := newTestEncryptor(t)
	ct, err := enc.Encrypt("")
	require.NoError(t, err)

	recovered, err := enc.Decrypt(ct)
	require.NoError(t, err)
	assert.Equal(t, "", recovered)
}

func TestEncryptDecrypt_LongPlaintext(t *testing.T) {
	enc := newTestEncryptor(t)
	plaintext := strings.Repeat("a", 4096)

	ct, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	recovered, err := enc.Decrypt(ct)
	require.NoError(t, err)
	assert.Equal(t, plaintext, recovered)
}
