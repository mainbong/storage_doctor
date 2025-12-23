package search

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/mainbong/storage_doctor/internal/httpclient"
)

// DuckDuckGoProvider implements search using DuckDuckGo HTML API
type DuckDuckGoProvider struct {
	client httpclient.HTTPClient
}

// NewDuckDuckGoProvider creates a new DuckDuckGo provider
func NewDuckDuckGoProvider() *DuckDuckGoProvider {
	return NewDuckDuckGoProviderWithClient(httpclient.NewDefaultHTTPClient())
}

// NewDuckDuckGoProviderWithClient creates a new DuckDuckGo provider with a custom HTTPClient (for testing)
func NewDuckDuckGoProviderWithClient(client httpclient.HTTPClient) *DuckDuckGoProvider {
	return &DuckDuckGoProvider{
		client: client,
	}
}

func (p *DuckDuckGoProvider) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	// DuckDuckGo Instant Answer API (limited, but free)
	// For better results, we'll use a simple HTML scraping approach
	// Note: This is a basic implementation. For production, consider using DuckDuckGo's official API
	
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))
	
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("duckduckgo API error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Simple HTML parsing (for production, use a proper HTML parser)
	results := p.parseHTMLResults(string(body), limit)
	
	return results, nil
}

func (p *DuckDuckGoProvider) parseHTMLResults(_ string, _ int) []SearchResult {
	var results []SearchResult
	
	// This is a simplified parser. For production, use goquery or similar
	// For now, we'll return empty results and suggest using a different provider
	// In a real implementation, you would parse the HTML properly
	
	return results
}

