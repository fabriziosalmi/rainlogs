package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fabriziosalmi/rainlogs/internal/auth"
	"github.com/fabriziosalmi/rainlogs/internal/models"
)

// ── API Key ───────────────────────────────────────────────────────────────────

func TestGenerateAPIKey_Format(t *testing.T) {
	plaintext, hash, prefix, err := auth.GenerateAPIKey()
	require.NoError(t, err)

	assert.True(t, len(plaintext) > 8, "plaintext must be non-trivial length")
	assert.NotEmpty(t, hash)
	assert.NotEmpty(t, prefix)
	assert.Contains(t, plaintext, "rl_", "key must carry rl_ prefix")
	assert.Len(t, prefix, 8, "lookup prefix must be 8 chars")
}

func TestGenerateAPIKey_Unique(t *testing.T) {
	k1, _, p1, err := auth.GenerateAPIKey()
	require.NoError(t, err)
	k2, _, p2, err := auth.GenerateAPIKey()
	require.NoError(t, err)

	assert.NotEqual(t, k1, k2, "generated keys must be unique")
	// Prefix collision is astronomically unlikely with 8 base64url chars.
	_ = p1
	_ = p2
}

func TestValidateAPIKey_Valid(t *testing.T) {
	plaintext, hash, _, err := auth.GenerateAPIKey()
	require.NoError(t, err)

	assert.True(t, auth.ValidateAPIKey(plaintext, hash))
}

func TestValidateAPIKey_Invalid(t *testing.T) {
	_, hash, _, err := auth.GenerateAPIKey()
	require.NoError(t, err)

	assert.False(t, auth.ValidateAPIKey("rl_wrongkey", hash))
}

func TestPrefixOf_Valid(t *testing.T) {
	plaintext, _, expectedPrefix, err := auth.GenerateAPIKey()
	require.NoError(t, err)

	prefix, err := auth.PrefixOf(plaintext)
	require.NoError(t, err)
	assert.Equal(t, expectedPrefix, prefix)
}

func TestPrefixOf_InvalidFormat(t *testing.T) {
	_, err := auth.PrefixOf("not-an-rl-key")
	require.Error(t, err)
}

func TestPrefixOf_TooShort(t *testing.T) {
	_, err := auth.PrefixOf("rl_short")
	require.Error(t, err)
}

// ── JWT ───────────────────────────────────────────────────────────────────────

func TestJWT_IssueAndVerify(t *testing.T) {
	secret := "super-secret-test-key-at-least-32-chars"
	customer := &models.Customer{
		ID:    uuid.New(),
		Name:  "Test Corp",
		Email: "test@example.com",
	}

	token, err := auth.IssueJWT(secret, customer, time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := auth.VerifyJWT(secret, token)
	require.NoError(t, err)
	assert.Equal(t, customer.ID.String(), claims.CustomerID)
}

func TestJWT_WrongSecret(t *testing.T) {
	customer := &models.Customer{ID: uuid.New()}
	token, err := auth.IssueJWT("correct-secret-key-that-is-long-enough", customer, time.Hour)
	require.NoError(t, err)

	_, err = auth.VerifyJWT("wrong-secret-key-that-is-long-enough!!", token)
	require.Error(t, err, "verification with wrong secret must fail")
}

func TestJWT_Expired(t *testing.T) {
	customer := &models.Customer{ID: uuid.New()}
	// Issue with -1 second TTL = already expired.
	token, err := auth.IssueJWT("secret-key-that-is-long-enough-for-test", customer, -time.Second)
	require.NoError(t, err)

	_, err = auth.VerifyJWT("secret-key-that-is-long-enough-for-test", token)
	require.Error(t, err, "expired token must fail verification")
}

func TestJWT_Tampered(t *testing.T) {
	customer := &models.Customer{ID: uuid.New()}
	token, err := auth.IssueJWT("my-jwt-signing-secret-key-32chars!!", customer, time.Hour)
	require.NoError(t, err)

	// Append garbage to the token signature.
	tampered := token + "tampered"
	_, err = auth.VerifyJWT("my-jwt-signing-secret-key-32chars!!", tampered)
	require.Error(t, err, "tampered token must fail verification")
}
