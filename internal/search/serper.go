package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mainbong/storage_doctor/internal/httpclient"
)

// SerperProvider implements search using Serper API
type SerperProvider struct {
	apiKey string
	client httpclient.HTTPClient
}

// NewSerperProvider creates a new Serper provider
func NewSerperProvider(apiKey string) *SerperProvider {
	return NewSerperProviderWithClient(apiKey, httpclient.NewDefaultHTTPClient())
}

// NewSerperProviderWithClient creates a new Serper provider with a custom HTTPClient (for testing)
func NewSerperProviderWithClient(apiKey string, client httpclient.HTTPClient) *SerperProvider {
	return &SerperProvider{
		apiKey: apiKey,
		client: client,
	}
}

func (p *SerperProvider) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("serper API key not set")
	}

	reqBody := map[string]interface{}{
		"q":   query,
		"num": limit,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://google.serper.dev/search", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("serper API error: %d - %s", resp.StatusCode, string(body))
	}

	var response struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]SearchResult, 0, len(response.Organic))
	for _, item := range response.Organic {
		results = append(results, SearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: item.Snippet,
		})
	}

	return results, nil
}
