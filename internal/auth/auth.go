package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/fabriziosalmi/rainlogs/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	tokenBytes  = 32
	tokenPrefix = "rl_"
	bcryptCost  = 12
	prefixLen   = 8 // chars of base64url used as O(1) lookup prefix
)

// GenerateAPIKey returns (plaintext, bcryptHash, lookupPrefix, error).
// The plaintext is shown to the user exactly once; only the hash is stored.
func GenerateAPIKey() (plaintext, hash, prefix string, err error) {
	b := make([]byte, tokenBytes)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", fmt.Errorf("auth: rand: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(b)
	if len(encoded) < prefixLen {
		return "", "", "", fmt.Errorf("auth: encoded key too short")
	}
	plaintext = tokenPrefix + encoded
	prefix = encoded[:prefixLen]
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcryptCost)
	if err != nil {
		return "", "", "", fmt.Errorf("auth: bcrypt: %w", err)
	}
	return plaintext, string(hashBytes), prefix, nil
}

// ValidateAPIKey compares a plaintext key against a bcrypt hash.
func ValidateAPIKey(plaintext, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)) == nil
}

// PrefixOf extracts the lookup prefix from a plaintext API key.
func PrefixOf(plaintext string) (string, error) {
	if !strings.HasPrefix(plaintext, tokenPrefix) {
		return "", fmt.Errorf("auth: invalid key format")
	}
	body := plaintext[len(tokenPrefix):]
	if len(body) < prefixLen {
		return "", fmt.Errorf("auth: key too short")
	}
	return body[:prefixLen], nil
}

// Claims is the JWT payload for internal service-to-service auth.
type Claims struct {
	CustomerID string `json:"cid"`
	jwt.RegisteredClaims
}

// IssueJWT signs a short-lived JWT for a customer.
func IssueJWT(secret string, c *models.Customer, ttl time.Duration) (string, error) {
	claims := Claims{
		CustomerID: c.ID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   c.ID.String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// VerifyJWT validates a JWT and returns the claims.
func VerifyJWT(secret, tokenStr string) (*Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth: jwt verify: %w", err)
	}
	return &claims, nil
}
