package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/mainbong/storage_doctor/internal/config"
	"github.com/mainbong/storage_doctor/internal/httpclient"
)

func TestAnthropicProvider_StreamChat(t *testing.T) {
	mockClient := httpclient.NewMockHTTPClient()
	cfg := &config.Config{
		Anthropic: struct {
			APIKey string `json:"api_key"`
			Model  string `json:"model"`
		}{
			APIKey: "test-key",
			Model:  "claude-haiku-4-5-20251001",
		},
	}

	// Mock SSE stream response
	streamData := `data: {"type":"message_start","message":{"id":"msg_123"}}
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" World"}}
data: {"type":"content_block_stop","index":0}
data: {"type":"message_stop"}
`

	mockClient.SetResponse("https://api.anthropic.com/v1/messages", 200, streamData, map[string]string{
		"Content-Type": "text/event-stream",
	})

	provider := NewAnthropicProviderWithClient(cfg, mockClient)

	var chunks []string
	err := provider.StreamChat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, func(chunk string) {
		chunks = append(chunks, chunk)
	})

	if err != nil {
		t.Fatalf("StreamChat() failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected chunks, got none")
	}
}

func TestAnthropicProvider_StreamChat_MissingAPIKey(t *testing.T) {
	cfg := &config.Config{
		Anthropic: struct {
			APIKey string `json:"api_key"`
			Model  string `json:"model"`
		}{
			APIKey: "",
			Model:  "claude-haiku-4-5-20251001",
		},
	}

	provider := NewAnthropicProvider(cfg)

	err := provider.StreamChat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, func(chunk string) {})

	if err == nil {
		t.Error("Expected error for missing API key, got nil")
	}
}

func TestAnthropicProvider_GetModel(t *testing.T) {
	cfg := &config.Config{
		Anthropic: struct {
			APIKey string `json:"api_key"`
			Model  string `json:"model"`
		}{
			APIKey: "test-key",
			Model:  "claude-haiku-4-5-20251001",
		},
	}

	provider := NewAnthropicProvider(cfg)

	if provider.GetModel() != "claude-haiku-4-5-20251001" {
		t.Errorf("Expected model 'claude-haiku-4-5-20251001', got '%s'", provider.GetModel())
	}
}

func TestAnthropicProvider_Chat(t *testing.T) {
	mockClient := httpclient.NewMockHTTPClient()
	cfg := &config.Config{
		Anthropic: struct {
			APIKey string `json:"api_key"`
			Model  string `json:"model"`
		}{
			APIKey: "test-key",
			Model:  "claude-haiku-4-5-20251001",
		},
	}

	// Note: Chat() uses StreamChat internally, so we need to mock the stream
	streamData := `data: {"type":"message_start"}
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" World"}}
data: {"type":"content_block_stop","index":0}
data: {"type":"message_stop"}
`
	mockClient.SetResponse("https://api.anthropic.com/v1/messages", 200, streamData, map[string]string{
		"Content-Type": "text/event-stream",
	})

	provider := NewAnthropicProviderWithClient(cfg, mockClient)

	response, err := provider.Chat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	})

	if err != nil {
		t.Fatalf("Chat() failed: %v", err)
	}

	if !strings.Contains(response, "Hello") {
		t.Errorf("Expected response to contain 'Hello', got '%s'", response)
	}
}

func TestAnthropicProvider_StreamChatWithTools_ToolCall(t *testing.T) {
	mockClient := httpclient.NewMockHTTPClient()
	cfg := &config.Config{
		Anthropic: struct {
			APIKey string `json:"api_key"`
			Model  string `json:"model"`
		}{
			APIKey: "test-key",
			Model:  "claude-haiku-4-5-20251001",
		},
	}

	streamData := `data: {"type":"message_start","message":{"id":"msg_123"}}
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"execute_command","input":{}}}
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"command\":\"ls\","}}
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"description\":\"list\"}"}}
data: {"type":"content_block_stop","index":0}
data: {"type":"message_stop"}
`

	mockClient.SetResponse("https://api.anthropic.com/v1/messages", 200, streamData, map[string]string{
		"Content-Type": "text/event-stream",
	})

	provider := NewAnthropicProviderWithClient(cfg, mockClient)

	var toolCalls []ToolCall
	err := provider.StreamChatWithTools(context.Background(), []Message{
		{Role: "user", Content: "run command"},
	}, []Tool{
		{
			Name:        "execute_command",
			Description: "test",
			InputSchema: map[string]interface{}{},
		},
	}, func(chunk string) {}, func(toolCall ToolCall) {
		toolCalls = append(toolCalls, toolCall)
	})

	if err != nil {
		t.Fatalf("StreamChatWithTools() failed: %v", err)
	}

	if len(toolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(toolCalls))
	}
	if toolCalls[0].Name != "execute_command" {
		t.Errorf("Expected tool name 'execute_command', got '%s'", toolCalls[0].Name)
	}
	if toolCalls[0].Input["command"] != "ls" {
		t.Errorf("Expected command 'ls', got '%v'", toolCalls[0].Input["command"])
	}
	if toolCalls[0].Input["description"] != "list" {
		t.Errorf("Expected description 'list', got '%v'", toolCalls[0].Input["description"])
	}
}
