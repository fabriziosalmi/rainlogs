package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GraphQLClient struct {
	baseURL    string // default "https://api.cloudflare.com/client/v4/graphql"
	httpClient *http.Client
	apiToken   string
}

func NewGraphQLClient(apiToken string) *GraphQLClient {
	return &GraphQLClient{
		baseURL:    "https://api.cloudflare.com/client/v4/graphql",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiToken:   apiToken,
	}
}

type FirewallEvent struct {
	Action             string    `json:"action"`
	ClientIP           string    `json:"clientIP"`
	ClientRequestPath  string    `json:"clientRequestPath"`
	ClientRequestQuery string    `json:"clientRequestQuery"`
	Datetime           time.Time `json:"datetime"`
	RayName            string    `json:"rayName"`
	RuleID             string    `json:"ruleId"`
	Source             string    `json:"source"`
	UserAgent          string    `json:"userAgent"`
}

type graphQLResponse struct {
	Data struct {
		Viewer struct {
			Zones []struct {
				FirewallEventsAdaptive []FirewallEvent `json:"firewallEventsAdaptive"`
			} `json:"zones"`
		} `json:"viewer"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (c *GraphQLClient) GetSecurityEvents(ctx context.Context, zoneID string, start, end time.Time) ([]FirewallEvent, error) {
	query := `query GetSecurityEvents($zoneTag: string, $start: Time!, $end: Time!) {
		viewer {
			zones(filter: { zoneTag: $zoneTag }) {
				firewallEventsAdaptive(
					filter: { datetime_geq: $start, datetime_leq: $end },
					limit: 1000,
					orderBy: [datetime_DESC]
				) {
					action
					clientIP
					clientRequestPath
					clientRequestQuery
					datetime
					rayName
					ruleId
					source
					userAgent
				}
			}
		}
	}`

	reqBody := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"zoneTag": zoneID,
			"start":   start.Format(time.RFC3339),
			"end":     end.Format(time.RFC3339),
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("graphql api error: %s", body)
	}

	var result graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", result.Errors[0].Message)
	}

	if len(result.Data.Viewer.Zones) == 0 {
		// Zone not found or no access
		return nil, nil
	}

	return result.Data.Viewer.Zones[0].FirewallEventsAdaptive, nil
}
