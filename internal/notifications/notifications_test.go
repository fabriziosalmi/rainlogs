package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlackNotifier_SendAlert(t *testing.T) {
	// 1. Setup Mock Server
	var capturedPayload slackPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/services/hooks/incoming-webhook", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		err := json.NewDecoder(r.Body).Decode(&capturedPayload)
		assert.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	// 2. Create Notifier
	notifier := NewSlackNotifier(server.URL + "/services/hooks/incoming-webhook")

	// 3. Test SendAlert
	ctx := context.Background()
	err := notifier.SendAlert(ctx, "proj-123", "critical", "Something went wrong")

	// 4. Assertions
	assert.NoError(t, err)
	assert.Equal(t, "Rainlogs Alert: Project proj-123", capturedPayload.Text)
	if assert.NotEmpty(t, capturedPayload.Attachments) {
		att := capturedPayload.Attachments[0]
		assert.Equal(t, "#ff0000", att.Color) // critical = red
		assert.Equal(t, "[critical] Alert", att.Title)
		assert.Equal(t, "Something went wrong", att.Text)
	}
}

func TestSlackNotifier_SendAlert_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(server.URL)
	err := notifier.SendAlert(context.Background(), "proj", "info", "msg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "slack api returned status: 500")
}
