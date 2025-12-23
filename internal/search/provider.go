package search

import (
	"context"
)

// SearchResult represents a search result
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// Provider is the interface for search providers
type Provider interface {
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
}

