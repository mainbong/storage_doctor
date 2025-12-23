package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/mainbong/storage_doctor/internal/config"
	"github.com/mainbong/storage_doctor/internal/httpclient"
)

func TestOpenAIProvider_StreamChat(t *testing.T) {
	mockClient := httpclient.NewMockHTTPClient()
	cfg := &config.Config{
		OpenAI: struct {
			APIKey string `json:"api_key"`
			Model  string `json:"model"`
		}{
			APIKey: "test-key",
			Model:  "gpt-5",
		},
	}

	// Mock SSE stream response
	streamData := `data: {"choices":[{"delta":{"content":"Hello"}}]}
data: {"choices":[{"delta":{"content":" World"}}]}
data: [DONE]
`

	mockClient.SetResponse("https://api.openai.com/v1/chat/completions", 200, streamData, map[string]string{
		"Content-Type": "text/event-stream",
	})

	provider := NewOpenAIProviderWithClient(cfg, mockClient)

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

func TestOpenAIProvider_StreamChat_MissingAPIKey(t *testing.T) {
	cfg := &config.Config{
		OpenAI: struct {
			APIKey string `json:"api_key"`
			Model  string `json:"model"`
		}{
			APIKey: "",
			Model:  "gpt-5",
		},
	}

	provider := NewOpenAIProvider(cfg)

	err := provider.StreamChat(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
	}, func(chunk string) {})

	if err == nil {
		t.Error("Expected error for missing API key, got nil")
	}
}

func TestOpenAIProvider_GetModel(t *testing.T) {
	cfg := &config.Config{
		OpenAI: struct {
			APIKey string `json:"api_key"`
			Model  string `json:"model"`
		}{
			APIKey: "test-key",
			Model:  "gpt-5",
		},
	}

	provider := NewOpenAIProvider(cfg)

	if provider.GetModel() != "gpt-5" {
		t.Errorf("Expected model 'gpt-5', got '%s'", provider.GetModel())
	}
}

func TestOpenAIProvider_Chat(t *testing.T) {
	mockClient := httpclient.NewMockHTTPClient()
	cfg := &config.Config{
		OpenAI: struct {
			APIKey string `json:"api_key"`
			Model  string `json:"model"`
		}{
			APIKey: "test-key",
			Model:  "gpt-5",
		},
	}

	// Mock SSE stream response
	streamData := `data: {"choices":[{"delta":{"content":"Hello"}}]}
data: {"choices":[{"delta":{"content":" World"}}]}
data: [DONE]
`

	mockClient.SetResponse("https://api.openai.com/v1/chat/completions", 200, streamData, map[string]string{
		"Content-Type": "text/event-stream",
	})

	provider := NewOpenAIProviderWithClient(cfg, mockClient)

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

func TestOpenAIProvider_StreamChatWithTools_ToolCall(t *testing.T) {
	mockClient := httpclient.NewMockHTTPClient()
	cfg := &config.Config{
		OpenAI: struct {
			APIKey string `json:"api_key"`
			Model  string `json:"model"`
		}{
			APIKey: "test-key",
			Model:  "gpt-5",
		},
	}

	streamData := `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"execute_command","arguments":"{\"command\":\"ls\","}}]}}]}
data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"description\":\"list\"}"}}]}}]}
data: [DONE]
`

	mockClient.SetResponse("https://api.openai.com/v1/chat/completions", 200, streamData, map[string]string{
		"Content-Type": "text/event-stream",
	})

	provider := NewOpenAIProviderWithClient(cfg, mockClient)

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
