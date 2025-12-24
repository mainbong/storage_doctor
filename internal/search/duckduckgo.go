package search

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/mainbong/storage_doctor/internal/httpclient"
	"golang.org/x/net/html"
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

func (p *DuckDuckGoProvider) parseHTMLResults(raw string, limit int) []SearchResult {
	var results []SearchResult

	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return results
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n == nil || (limit > 0 && len(results) >= limit) {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" && hasClass(n, "result__a") {
			title := strings.TrimSpace(nodeText(n))
			url := strings.TrimSpace(getAttr(n, "href"))
			snippet := ""
			if parent := findAncestorWithClass(n, "result"); parent != nil {
				if snippetNode := findDescendantWithClass(parent, "result__snippet"); snippetNode != nil {
					snippet = strings.TrimSpace(nodeText(snippetNode))
				}
			}
			if title != "" || url != "" {
				results = append(results, SearchResult{
					Title:   title,
					URL:     url,
					Snippet: snippet,
				})
			}
			if limit > 0 && len(results) >= limit {
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
			if limit > 0 && len(results) >= limit {
				return
			}
		}
	}
	walk(doc)
	return results
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key != "class" {
			continue
		}
		for _, part := range strings.Fields(attr.Val) {
			if part == class {
				return true
			}
		}
	}
	return false
}

func nodeText(n *html.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
			return
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func findAncestorWithClass(n *html.Node, class string) *html.Node {
	for cur := n.Parent; cur != nil; cur = cur.Parent {
		if cur.Type == html.ElementNode && hasClass(cur, class) {
			return cur
		}
	}
	return nil
}

func findDescendantWithClass(n *html.Node, class string) *html.Node {
	if n == nil {
		return nil
	}
	var found *html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil || found != nil {
			return
		}
		if node.Type == html.ElementNode && hasClass(node, class) {
			found = node
			return
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
			if found != nil {
				return
			}
		}
	}
	walk(n)
	return found
}
