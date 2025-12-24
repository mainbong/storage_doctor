package llm

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RateLimiter struct {
	window      time.Duration
	maxTokens   int
	maxRequests int

	mu           sync.Mutex
	windowStart  time.Time
	usedTokens   int
	usedRequests int
}

type RateLimitReporter func(wait time.Duration, waiting bool)

type rateLimitReporterKey struct{}

func WithRateLimitReporter(ctx context.Context, reporter RateLimitReporter) context.Context {
	if reporter == nil {
		return ctx
	}
	return context.WithValue(ctx, rateLimitReporterKey{}, reporter)
}

func reportRateLimit(ctx context.Context, wait time.Duration, waiting bool) {
	if ctx == nil {
		return
	}
	if reporter, ok := ctx.Value(rateLimitReporterKey{}).(RateLimitReporter); ok && reporter != nil {
		reporter(wait, waiting)
	}
}

func NewRateLimiter(window time.Duration, maxTokens, maxRequests int) *RateLimiter {
	return &RateLimiter{
		window:      window,
		maxTokens:   maxTokens,
		maxRequests: maxRequests,
	}
}

func (l *RateLimiter) Wait(ctx context.Context, tokens int) error {
	if l == nil {
		return nil
	}
	if tokens < 0 {
		tokens = 0
	}
	for {
		wait := l.reserve(tokens)
		if wait == 0 {
			reportRateLimit(ctx, 0, false)
			return nil
		}
		reportRateLimit(ctx, wait, true)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			reportRateLimit(ctx, 0, false)
			return ctx.Err()
		case <-timer.C:
		}
		reportRateLimit(ctx, 0, false)
	}
}

func (l *RateLimiter) reserve(tokens int) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if l.windowStart.IsZero() || now.Sub(l.windowStart) >= l.window {
		l.windowStart = now
		l.usedTokens = 0
		l.usedRequests = 0
	}

	tokensOK := l.maxTokens <= 0 || l.usedTokens+tokens <= l.maxTokens
	requestsOK := l.maxRequests <= 0 || l.usedRequests+1 <= l.maxRequests
	if tokensOK && requestsOK {
		l.usedTokens += tokens
		l.usedRequests++
		return 0
	}

	wait := l.windowStart.Add(l.window).Sub(now)
	if wait < 0 {
		return 0
	}
	return wait
}

func (l *RateLimiter) UpdateLimits(tokensPerWindow, requestsPerWindow int) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if tokensPerWindow > 0 {
		l.maxTokens = tokensPerWindow
	}
	if requestsPerWindow > 0 {
		l.maxRequests = requestsPerWindow
	}
}

func updateLimiterFromHeaders(l *RateLimiter, headers http.Header, tokensKey, requestsKey string) {
	if l == nil {
		return
	}
	l.UpdateLimits(parseRateLimitHeader(headers.Get(tokensKey)), parseRateLimitHeader(headers.Get(requestsKey)))
}

func parseRateLimitHeader(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if strings.Contains(value, ".") {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return int(f)
		}
	}
	if n, err := strconv.Atoi(value); err == nil {
		return n
	}
	return 0
}

func defaultRateLimits(provider string) (tokensPerMinute, requestsPerMinute int) {
	switch strings.ToLower(provider) {
	case "openai":
		return 50000, 60
	case "anthropic":
		return 50000, 60
	default:
		return 50000, 60
	}
}
