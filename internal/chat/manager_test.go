package chat

import (
	"context"
	"errors"
	"testing"

	"github.com/mainbong/storage_doctor/internal/llm"
)

func TestNewManager(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	manager := NewManager(mockProvider)

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.provider != mockProvider {
		t.Error("Expected provider to be set, but it's different")
	}

	if manager.maxMessages != 50 {
		t.Errorf("Expected maxMessages 50, got %d", manager.maxMessages)
	}

	if manager.summarizeThreshold != 30 {
		t.Errorf("Expected summarizeThreshold 30, got %d", manager.summarizeThreshold)
	}
}

func TestAddMessage(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	manager := NewManager(mockProvider)

	manager.AddMessage("user", "Hello")
	manager.AddMessage("assistant", "Hi there")

	messages := manager.GetMessages()
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0].Role != "user" {
		t.Errorf("Expected first message role 'user', got '%s'", messages[0].Role)
	}

	if messages[0].Content != "Hello" {
		t.Errorf("Expected first message content 'Hello', got '%s'", messages[0].Content)
	}
}

func TestGetMessages(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	manager := NewManager(mockProvider)

	manager.AddMessage("user", "Test")
	messages := manager.GetMessages()

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}
}

func TestStreamChat(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	mockProvider.SetStreamChunks([]string{"Hello", " World"})
	manager := NewManager(mockProvider)

	var chunks []string
	err := manager.StreamChat(context.Background(), "test", func(chunk string) {
		chunks = append(chunks, chunk)
	})

	if err != nil {
		t.Fatalf("StreamChat() failed: %v", err)
	}

	if len(chunks) != 2 {
		t.Errorf("Expected 2 chunks, got %d", len(chunks))
	}

	// Verify message was added
	messages := manager.GetMessages()
	if len(messages) < 2 {
		t.Error("Expected user and assistant messages to be added")
	}
}

func TestStreamChatWithTools(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	mockProvider.SetStreamChunks([]string{"I'll help"})
	mockProvider.SetToolCalls([]llm.ToolCall{
		{ID: "call_1", Name: "execute_command", Input: map[string]interface{}{"command": "ls"}},
	})
	manager := NewManager(mockProvider)

	var chunks []string
	var toolCalls []llm.ToolCall

	tools := []llm.Tool{
		{Name: "execute_command", Description: "Execute a command"},
	}

	err := manager.StreamChatWithTools(context.Background(), "test", tools, func(chunk string) {
		chunks = append(chunks, chunk)
	}, func(toolCall llm.ToolCall) {
		toolCalls = append(toolCalls, toolCall)
	})

	if err != nil {
		t.Fatalf("StreamChatWithTools() failed: %v", err)
	}

	if len(toolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(toolCalls))
	}

	if toolCalls[0].Name != "execute_command" {
		t.Errorf("Expected tool call name 'execute_command', got '%s'", toolCalls[0].Name)
	}
}

func TestChat(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	mockProvider.SetStreamChunks([]string{"Hello", " World"})
	manager := NewManager(mockProvider)

	response, err := manager.Chat(context.Background(), "test")
	if err != nil {
		t.Fatalf("Chat() failed: %v", err)
	}

	if response != "Hello World" {
		t.Errorf("Expected response 'Hello World', got '%s'", response)
	}
}

func TestClear(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	manager := NewManager(mockProvider)

	manager.AddMessage("user", "test")
	manager.Clear()

	messages := manager.GetMessages()
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages after Clear(), got %d", len(messages))
	}
}

func TestSetSystemPrompt(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	manager := NewManager(mockProvider)

	manager.AddMessage("user", "test")
	manager.SetSystemPrompt("You are a helpful assistant")

	messages := manager.GetMessages()
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages (system + user), got %d", len(messages))
	}

	if messages[0].Role != "system" {
		t.Errorf("Expected first message to be system, got '%s'", messages[0].Role)
	}

	if messages[0].Content != "You are a helpful assistant" {
		t.Errorf("Expected system prompt 'You are a helpful assistant', got '%s'", messages[0].Content)
	}
}

func TestSetSystemPrompt_ReplaceExisting(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	manager := NewManager(mockProvider)

	manager.SetSystemPrompt("Old prompt")
	manager.AddMessage("user", "test")
	manager.SetSystemPrompt("New prompt")

	messages := manager.GetMessages()
	systemCount := 0
	for _, msg := range messages {
		if msg.Role == "system" {
			systemCount++
			if msg.Content != "New prompt" {
				t.Errorf("Expected system prompt 'New prompt', got '%s'", msg.Content)
			}
		}
	}

	if systemCount != 1 {
		t.Errorf("Expected 1 system message, got %d", systemCount)
	}
}

func TestStreamChat_ProviderError(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	mockProvider.SetStreamError(errors.New("provider error"))
	manager := NewManager(mockProvider)

	err := manager.StreamChat(context.Background(), "test", func(chunk string) {})
	if err == nil {
		t.Error("Expected error from provider, got nil")
	}
}

