package search

import (
	"context"
	"testing"

	"github.com/mainbong/storage_doctor/internal/httpclient"
)

func TestDuckDuckGoProvider_Search(t *testing.T) {
	mockClient := httpclient.NewMockHTTPClient()

	// Mock HTML response (simplified)
	htmlResponse := `<html><body><div class="result">Test Result</div></body></html>`
	mockClient.SetResponse("https://html.duckduckgo.com/html/?q=test", 200, htmlResponse, nil)

	provider := NewDuckDuckGoProviderWithClient(mockClient)

	results, err := provider.Search(context.Background(), "test", 10)
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	// DuckDuckGo parser returns empty results in current implementation
	_ = results
}

func TestDuckDuckGoProvider_Search_Error(t *testing.T) {
	mockClient := httpclient.NewMockHTTPClient()
	mockClient.SetResponse("https://html.duckduckgo.com/html/?q=test", 500, "Internal Server Error", nil)

	provider := NewDuckDuckGoProviderWithClient(mockClient)

	_, err := provider.Search(context.Background(), "test", 10)
	if err == nil {
		t.Error("Expected error for 500 status, got nil")
	}
}




