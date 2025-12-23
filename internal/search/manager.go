package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/mainbong/storage_doctor/internal/config"
)

// Manager manages web search operations
type Manager struct {
	provider Provider
}

// NewManager creates a new search manager
func NewManager(cfg *config.Config) (*Manager, error) {
	var provider Provider

	switch cfg.Search.Provider {
	case "google":
		if cfg.Search.Google.APIKey == "" || cfg.Search.Google.CX == "" {
			return nil, fmt.Errorf("google search requires API key and CX")
		}
		provider = NewGoogleProvider(cfg.Search.Google.APIKey, cfg.Search.Google.CX)
	case "serper":
		if cfg.Search.Serper.APIKey == "" {
			return nil, fmt.Errorf("serper search requires API key")
		}
		provider = NewSerperProvider(cfg.Search.Serper.APIKey)
	case "duckduckgo":
		provider = NewDuckDuckGoProvider()
	default:
		return nil, fmt.Errorf("unknown search provider: %s", cfg.Search.Provider)
	}

	return &Manager{
		provider: provider,
	}, nil
}

// Search performs a web search
func (m *Manager) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	return m.provider.Search(ctx, query, limit)
}

// FormatResults formats search results as a string for LLM context
func (m *Manager) FormatResults(results []SearchResult) string {
	if len(results) == 0 {
		return "검색 결과가 없습니다."
	}

	var builder strings.Builder
	builder.WriteString("웹 검색 결과:\n\n")

	for i, result := range results {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
		builder.WriteString(fmt.Sprintf("   URL: %s\n", result.URL))
		builder.WriteString(fmt.Sprintf("   요약: %s\n\n", result.Snippet))
	}

	return builder.String()
}
