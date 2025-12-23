package llm

import (
	"fmt"

	"github.com/mainbong/storage_doctor/internal/config"
)

// NewProvider creates a new LLM provider based on configuration
func NewProvider(cfg *config.Config) (Provider, error) {
	switch cfg.LLMProvider {
	case "anthropic":
		return NewAnthropicProvider(cfg), nil
	case "openai":
		return NewOpenAIProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.LLMProvider)
	}
}

