package cloudflare

import (
	"fmt"
	"strconv"
	"time"
)

// RateLimitError is returned when Cloudflare responds with HTTP 429.
type RateLimitError struct {
	RetryAfter time.Duration
	Message    string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited: %s (retry after %v)", e.Message, e.RetryAfter)
}

// ParseRetryAfter parses the Retry-After header.
// It supports both seconds (integer) and HTTP date format.
func ParseRetryAfter(header string) time.Duration {
	if header == "" {
		return 30 * time.Second // Default backoff
	}

	// Try parsing as integer seconds
	if seconds, err := strconv.Atoi(header); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP date
	if t, err := time.Parse(time.RFC1123, header); err == nil {
		return time.Until(t)
	}

	return 30 * time.Second
}
