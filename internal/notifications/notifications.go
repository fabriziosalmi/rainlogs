package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type NotificationService interface {
	SendAlert(ctx context.Context, projectID string, severity string, message string) error
}

type ConsoleNotifier struct{}

func (n *ConsoleNotifier) SendAlert(_ context.Context, projectID, severity, message string) error {
	fmt.Printf("[ALERT][%s] Project: %s - %s\n", severity, projectID, message)
	return nil
}

// SlackNotifier sends alerts to a Slack Incoming Webhook.
type SlackNotifier struct {
	WebhookURL string
	HTTPClient *http.Client
}

func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		WebhookURL: webhookURL,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
}

type slackPayload struct {
	Text        string       `json:"text"`
	Attachments []attachment `json:"attachments,omitempty"`
}

type attachment struct {
	Color  string `json:"color"`
	Title  string `json:"title"`
	Text   string `json:"text"`
	Footer string `json:"footer"`
	Ts     int64  `json:"ts"`
}

func (n *SlackNotifier) SendAlert(ctx context.Context, projectID, severity, message string) error {
	if n.WebhookURL == "" {
		return nil
	}

	color := "#36a64f" // Green
	if severity == "error" || severity == "critical" {
		color = "#ff0000" // Red
	} else if severity == "warning" {
		color = "#ffcc00" // Yellow
	}

	payload := slackPayload{
		Text: fmt.Sprintf("Rainlogs Alert: Project %s", projectID),
		Attachments: []attachment{
			{
				Color:  color,
				Title:  fmt.Sprintf("[%s] Alert", severity),
				Text:   message,
				Footer: "Rainlogs Worker",
				Ts:     time.Now().Unix(),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.WebhookURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("send slack alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack api returned status: %d", resp.StatusCode)
	}

	return nil
}
