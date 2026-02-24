package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Config for integration tests.
const (
	BaseURL = "http://localhost:8080"
)

func TestHealthCheck(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, BaseURL+"/health", http.NoBody)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCustomerLifecycle(t *testing.T) {
	// 1. Create Customer
	customerID := createCustomer(t)

	// 2. Get Customer
	getCustomer(t, customerID)
}

func createCustomer(t *testing.T) string {
	payload := map[string]interface{}{
		"name":           "Integration Test Corp",
		"email":          fmt.Sprintf("test+%d@example.com", time.Now().UnixNano()),
		"cf_account_id":  "cf_acc_1234567890",
		"cf_api_key":     "cf_key_secret_123",
		"retention_days": 30,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, BaseURL+"/customers", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// print body for debugging
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body) //nolint:errcheck // best-effort debug output
		t.Fatalf("expected 201 Created, got %d: %s", resp.StatusCode, buf.String())
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	id, ok := result["id"].(string)
	require.True(t, ok, "response should contain customer id")
	return id
}

func getCustomer(t *testing.T, id string) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, BaseURL+"/customers/"+id, http.NoBody)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestMain(m *testing.M) {
	// Optional: Check if service is up before running tests
	if err := waitForService(BaseURL + "/health"); err != nil {
		fmt.Printf("Skipping integration tests: service not available at %s\n", BaseURL)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func waitForService(url string) error {
	for i := 0; i < 30; i++ {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("service not reachable")
}
