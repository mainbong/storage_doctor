package llm

import (
	"context"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"` // "user", "assistant", "system"
	Content string `json:"content"`
}

// Provider is the interface for LLM providers
type Provider interface {
	// StreamChat streams a chat completion
	StreamChat(ctx context.Context, messages []Message, onChunk func(string)) error

	// StreamChatWithTools streams chat with tool support
	StreamChatWithTools(ctx context.Context, messages []Message, tools []Tool, onChunk func(string), onToolCall func(ToolCall)) error

	// Chat sends a chat completion request and returns the full response
	Chat(ctx context.Context, messages []Message) (string, error)

	// GetModel returns the model name being used
	GetModel() string
}
