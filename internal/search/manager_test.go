package search

import (
	"strings"
	"testing"

	"github.com/mainbong/storage_doctor/internal/config"
)

func TestFormatResults(t *testing.T) {
	manager := &Manager{}

	results := []SearchResult{
		{Title: "Test 1", URL: "https://example.com/1", Snippet: "Snippet 1"},
		{Title: "Test 2", URL: "https://example.com/2", Snippet: "Snippet 2"},
	}

	formatted := manager.FormatResults(results)
	if formatted == "" {
		t.Error("Expected formatted results, got empty string")
	}

	if !strings.Contains(formatted, "Test 1") {
		t.Error("Expected formatted results to contain 'Test 1'")
	}
	if !strings.Contains(formatted, "https://example.com/1") {
		t.Error("Expected formatted results to contain URL")
	}
}

func TestFormatResults_Empty(t *testing.T) {
	manager := &Manager{}

	formatted := manager.FormatResults([]SearchResult{})
	if formatted != "검색 결과가 없습니다." {
		t.Errorf("Expected '검색 결과가 없습니다.', got '%s'", formatted)
	}
}

func TestNewManager_DuckDuckGo(t *testing.T) {
	cfg := &config.Config{
		Search: struct {
			Provider string `json:"provider"`
			Google   struct {
				APIKey string `json:"api_key"`
				CX     string `json:"cx"`
			} `json:"google"`
			Bing struct {
				APIKey string `json:"api_key"`
			} `json:"bing"`
			Serper struct {
				APIKey string `json:"api_key"`
			} `json:"serper"`
		}{
			Provider: "duckduckgo",
		},
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestNewManager_Google(t *testing.T) {
	cfg := &config.Config{
		Search: struct {
			Provider string `json:"provider"`
			Google   struct {
				APIKey string `json:"api_key"`
				CX     string `json:"cx"`
			} `json:"google"`
			Bing struct {
				APIKey string `json:"api_key"`
			} `json:"bing"`
			Serper struct {
				APIKey string `json:"api_key"`
			} `json:"serper"`
		}{
			Provider: "google",
			Google: struct {
				APIKey string `json:"api_key"`
				CX     string `json:"cx"`
			}{
				APIKey: "test-key",
				CX:     "test-cx",
			},
		},
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestNewManager_Google_MissingKeys(t *testing.T) {
	cfg := &config.Config{
		Search: struct {
			Provider string `json:"provider"`
			Google   struct {
				APIKey string `json:"api_key"`
				CX     string `json:"cx"`
			} `json:"google"`
			Bing struct {
				APIKey string `json:"api_key"`
			} `json:"bing"`
			Serper struct {
				APIKey string `json:"api_key"`
			} `json:"serper"`
		}{
			Provider: "google",
		},
	}

	_, err := NewManager(cfg)
	if err == nil {
		t.Error("Expected error for missing Google API keys, got nil")
	}
}

func TestNewManager_Serper(t *testing.T) {
	cfg := &config.Config{
		Search: struct {
			Provider string `json:"provider"`
			Google   struct {
				APIKey string `json:"api_key"`
				CX     string `json:"cx"`
			} `json:"google"`
			Bing struct {
				APIKey string `json:"api_key"`
			} `json:"bing"`
			Serper struct {
				APIKey string `json:"api_key"`
			} `json:"serper"`
		}{
			Provider: "serper",
			Serper: struct {
				APIKey string `json:"api_key"`
			}{
				APIKey: "test-key",
			},
		},
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestNewManager_Serper_MissingKey(t *testing.T) {
	cfg := &config.Config{
		Search: struct {
			Provider string `json:"provider"`
			Google   struct {
				APIKey string `json:"api_key"`
				CX     string `json:"cx"`
			} `json:"google"`
			Bing struct {
				APIKey string `json:"api_key"`
			} `json:"bing"`
			Serper struct {
				APIKey string `json:"api_key"`
			} `json:"serper"`
		}{
			Provider: "serper",
		},
	}

	_, err := NewManager(cfg)
	if err == nil {
		t.Error("Expected error for missing Serper API key, got nil")
	}
}

func TestNewManager_UnknownProvider(t *testing.T) {
	cfg := &config.Config{
		Search: struct {
			Provider string `json:"provider"`
			Google   struct {
				APIKey string `json:"api_key"`
				CX     string `json:"cx"`
			} `json:"google"`
			Bing struct {
				APIKey string `json:"api_key"`
			} `json:"bing"`
			Serper struct {
				APIKey string `json:"api_key"`
			} `json:"serper"`
		}{
			Provider: "unknown",
		},
	}

	_, err := NewManager(cfg)
	if err == nil {
		t.Error("Expected error for unknown provider, got nil")
	}
}

