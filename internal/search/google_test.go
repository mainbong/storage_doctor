package search

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mainbong/storage_doctor/internal/httpclient"
)

func TestGoogleProvider_Search(t *testing.T) {
	mockClient := httpclient.NewMockHTTPClient()

	// Mock successful response
	response := map[string]interface{}{
		"items": []map[string]interface{}{
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
	mockClient.SetResponse("https://www.googleapis.com/customsearch/v1?key=test-key&cx=test-cx&q=test&num=10", 200, string(responseBody), nil)

	provider := NewGoogleProviderWithClient("test-key", "test-cx", mockClient)

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

func TestGoogleProvider_Search_Error(t *testing.T) {
	mockClient := httpclient.NewMockHTTPClient()
	mockClient.SetResponse("https://www.googleapis.com/customsearch/v1?key=test-key&cx=test-cx&q=test&num=10", 400, "Bad Request", nil)

	provider := NewGoogleProviderWithClient("test-key", "test-cx", mockClient)

	_, err := provider.Search(context.Background(), "test", 10)
	if err == nil {
		t.Error("Expected error for 400 status, got nil")
	}
}

func TestGoogleProvider_Search_MissingKeys(t *testing.T) {
	provider := NewGoogleProvider("", "")

	_, err := provider.Search(context.Background(), "test", 10)
	if err == nil {
		t.Error("Expected error for missing API keys, got nil")
	}
}

