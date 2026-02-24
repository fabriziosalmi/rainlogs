// Package cloudflare provides a Cloudflare Logpull API client.
//
// Key constraints:
//   - Logs are available with a minimum 1-minute delay.
//   - Logs are retained by Cloudflare for 7 days (max Logpull window).
//   - Maximum pull window per request: 1 hour.
//   - RainLogs must pull before CF retention expires.
//   - Enterprise customers should use Logpush instead.
package cloudflare

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fabriziosalmi/rainlogs/internal/config"
)

// ErrRateLimited is returned when Cloudflare responds with HTTP 429.
// Workers should treat it as a retriable/transient error.
var ErrRateLimited = errors.New("cloudflare: rate limited")

const defaultBaseURL = "https://api.cloudflare.com/client/v4"

// Client is a Cloudflare Logpull API client for a single zone.
type Client struct {
	baseURL    string
	httpClient *http.Client
	zoneID     string
	apiKey     string
}

// NewClient creates a Client for a specific zone.
func NewClient(cfg config.CloudflareConfig, zoneID, apiKey string) *Client {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	return &Client{
		baseURL:    base,
		httpClient: &http.Client{Timeout: cfg.RequestTimeout},
		zoneID:     zoneID,
		apiKey:     apiKey,
	}
}

// PullLogs fetches NDJSON log lines for [from, to) (max 1-hour window).
// Returns raw decompressed NDJSON bytes.
func (c *Client) PullLogs(ctx context.Context, from, to time.Time, fields []string) ([]byte, error) {
	if to.Sub(from) > time.Hour {
		return nil, fmt.Errorf("cloudflare: window exceeds 1 hour")
	}
	if time.Since(to) < time.Minute {
		return nil, fmt.Errorf("cloudflare: logs not yet available (min 1-min delay)")
	}

	u, err := url.Parse(fmt.Sprintf("%s/zones/%s/logs/received", c.baseURL, c.zoneID))
	if err != nil {
		return nil, fmt.Errorf("cloudflare: parse url: %w", err)
	}
	q := u.Query()
	q.Set("start", from.UTC().Format(time.RFC3339))
	q.Set("end", to.UTC().Format(time.RFC3339))
	q.Set("timestamps", "rfc3339")
	if len(fields) > 0 {
		q.Set("fields", strings.Join(fields, ","))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfterStr := resp.Header.Get("Retry-After")
		delay := 30 * time.Second
		if val, err := strconv.Atoi(retryAfterStr); err == nil {
			delay = time.Duration(val) * time.Second
		}
		return nil, &RateLimitError{
			Message:    "Cloudflare 429",
			RetryAfter: delay,
		}
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("cloudflare: HTTP %d: %s", resp.StatusCode, body)
	}

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, gzErr := gzip.NewReader(resp.Body)
		if gzErr != nil {
			return nil, fmt.Errorf("cloudflare: gzip reader: %w", gzErr)
		}
		defer gr.Close()
		reader = gr
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: read body: %w", err)
	}
	return data, nil
}
