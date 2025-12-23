package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/mainbong/storage_doctor/internal/httpclient"
)

// GoogleProvider implements search using Google Custom Search API
type GoogleProvider struct {
	apiKey string
	cx     string
	client httpclient.HTTPClient
}

// NewGoogleProvider creates a new Google provider
func NewGoogleProvider(apiKey, cx string) *GoogleProvider {
	return NewGoogleProviderWithClient(apiKey, cx, httpclient.NewDefaultHTTPClient())
}

// NewGoogleProviderWithClient creates a new Google provider with a custom HTTPClient (for testing)
func NewGoogleProviderWithClient(apiKey, cx string, client httpclient.HTTPClient) *GoogleProvider {
	return &GoogleProvider{
		apiKey: apiKey,
		cx:     cx,
		client: client,
	}
}

func (p *GoogleProvider) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if p.apiKey == "" || p.cx == "" {
		return nil, fmt.Errorf("google API key or CX not set")
	}

	searchURL := fmt.Sprintf(
		"https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s&num=%d",
		url.QueryEscape(p.apiKey),
		url.QueryEscape(p.cx),
		url.QueryEscape(query),
		limit,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google API error: %d - %s", resp.StatusCode, string(body))
	}

	var response struct {
		Items []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]SearchResult, 0, len(response.Items))
	for _, item := range response.Items {
		results = append(results, SearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: item.Snippet,
		})
	}

	return results, nil
}
