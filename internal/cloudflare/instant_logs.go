package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type InstantLogsClient struct {
	apiToken string
	zoneID   string
	baseURL  string
}

func NewInstantLogsClient(apiToken, zoneID string) *InstantLogsClient {
	return &InstantLogsClient{
		apiToken: apiToken,
		zoneID:   zoneID,
		baseURL:  "https://api.cloudflare.com/client/v4",
	}
}

// StartSession creates a job and returns the WebSocket URL.
func (c *InstantLogsClient) StartSession(ctx context.Context) (string, error) {
	// 1. Create Job
	url := fmt.Sprintf("%s/zones/%s/logpush/edge/jobs", c.baseURL, c.zoneID)
	payload := `{"kind":"instant-logs","fields":"ClientIP,EdgeStartTimestamp,ClientRequestURI,ClientRequestMethod,EdgeResponseStatus,ClientRequestUserAgent,RayID"}`

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create intent request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do intent request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("create job failed: HTTP %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Result struct {
			DestinationConf string `json:"destination_conf"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode intent response: %w", err)
	}

	return result.Result.DestinationConf, nil
}

// Stream logs from WebSocket. Returns a channel of raw JSON bytes.
func (c *InstantLogsClient) Stream(ctx context.Context, wsURL string) (<-chan []byte, error) {
	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.DialContext(ctx, wsURL, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}

	ch := make(chan []byte, 100)

	go func() {
		defer close(ch)
		defer conn.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, msg, err := conn.ReadMessage()
				if err != nil {
					// Check if closed
					return
				}
				// Instant logs sends newline/comma separated entries or single entry?
				// Usually single JSON object per message.
				ch <- msg
			}
		}
	}()

	return ch, nil
}
