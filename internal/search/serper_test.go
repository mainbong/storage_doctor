package search

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mainbong/storage_doctor/internal/httpclient"
)

func TestSerperProvider_Search(t *testing.T) {
	mockClient := httpclient.NewMockHTTPClient()

	// Mock successful response
	response := map[string]interface{}{
		"organic": []map[string]interface{}{
			{
				"title":   "Test Result 1",
				"link":    "https://example.com/1",
				"snippet": "Snippet 1",
			},
			{
				"title":   "Test Result 2",
				"link":    "https://example.com/2",
				"snippet": "Snippet 2",
			},
		},
	}

	responseBody, _ := json.Marshal(response)
	mockClient.SetResponse("https://google.serper.dev/search", 200, string(responseBody), map[string]string{
		"Content-Type": "application/json",
	})

	provider := NewSerperProviderWithClient("test-key", mockClient)

	results, err := provider.Search(context.Background(), "test", 10)
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if results[0].Title != "Test Result 1" {
		t.Errorf("Expected title 'Test Result 1', got '%s'", results[0].Title)
	}
}

func TestSerperProvider_Search_Error(t *testing.T) {
	mockClient := httpclient.NewMockHTTPClient()
	mockClient.SetResponse("https://google.serper.dev/search", 401, "Unauthorized", nil)

	provider := NewSerperProviderWithClient("test-key", mockClient)

	_, err := provider.Search(context.Background(), "test", 10)
	if err == nil {
		t.Error("Expected error for 401 status, got nil")
	}
}

func TestSerperProvider_Search_MissingKey(t *testing.T) {
	provider := NewSerperProvider("")

	_, err := provider.Search(context.Background(), "test", 10)
	if err == nil {
		t.Error("Expected error for missing API key, got nil")
	}
}

