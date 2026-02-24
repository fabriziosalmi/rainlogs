package notifications

import (
	"context"
	"fmt"
)

type NotificationService interface {
	SendAlert(ctx context.Context, projectID string, severity string, message string) error
}

type ConsoleNotifier struct{}

func (n *ConsoleNotifier) SendAlert(_ context.Context, projectID, severity, message string) error {
	fmt.Printf("[ALERT][%s] Project: %s - %s\n", severity, projectID, message)
	return nil
}

// SlackNotifier stub for future implementation.
type SlackNotifier struct {
	WebhookURL string
}

func (n *SlackNotifier) SendAlert(_ context.Context, projectID string, severity string, message string) error {
	// Real implementation would POST to Slack
	return nil
}
