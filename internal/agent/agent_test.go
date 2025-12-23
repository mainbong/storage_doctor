package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mainbong/storage_doctor/internal/chat"
	"github.com/mainbong/storage_doctor/internal/filesystem"
	"github.com/mainbong/storage_doctor/internal/llm"
)

func TestNewAgent(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	mockChatManager := chat.NewManager(mockProvider)
	mockFS := filesystem.NewMockFileSystem()
	mockSkillManager, _ := NewSkillManagerWithFS("/test/skills", mockFS)

	agentInstance := NewAgent(mockProvider, mockChatManager, mockSkillManager)

	if agentInstance == nil {
		t.Fatal("NewAgent() returned nil")
	}

	if agentInstance.llmProvider != mockProvider {
		t.Error("Expected llmProvider to be set")
	}

	if agentInstance.chatManager != mockChatManager {
		t.Error("Expected chatManager to be set")
	}

	if agentInstance.skillManager != mockSkillManager {
		t.Error("Expected skillManager to be set")
	}
}

func TestStreamTask(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	mockProvider.SetStreamChunks([]string{"I'll help you"})
	mockChatManager := chat.NewManager(mockProvider)
	mockFS := filesystem.NewMockFileSystem()
	mockSkillManager, _ := NewSkillManagerWithFS("/test/skills", mockFS)

	agentInstance := NewAgent(mockProvider, mockChatManager, mockSkillManager)

	var chunks []string
	err := agentInstance.StreamTask(context.Background(), "test task", func(chunk string) {
		chunks = append(chunks, chunk)
	}, func(toolCall llm.ToolCall) (string, error) {
		// Do nothing
		return "", nil
	})

	if err != nil {
		t.Fatalf("StreamTask() failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected chunks, got none")
	}
}

func TestStreamTask_ProviderError(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	mockProvider.SetStreamError(errors.New("provider error"))
	mockChatManager := chat.NewManager(mockProvider)
	mockFS := filesystem.NewMockFileSystem()
	mockSkillManager, _ := NewSkillManagerWithFS("/test/skills", mockFS)

	agentInstance := NewAgent(mockProvider, mockChatManager, mockSkillManager)

	err := agentInstance.StreamTask(context.Background(), "test task", func(chunk string) {}, func(toolCall llm.ToolCall) (string, error) {
		return "", nil
	})
	if err == nil {
		t.Error("Expected error from provider, got nil")
	}
}

func TestStreamTask_EmptyResponse(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	mockProvider.SetStreamChunks([]string{}) // Empty response
	mockChatManager := chat.NewManager(mockProvider)
	mockFS := filesystem.NewMockFileSystem()
	mockSkillManager, _ := NewSkillManagerWithFS("/test/skills", mockFS)

	agentInstance := NewAgent(mockProvider, mockChatManager, mockSkillManager)

	err := agentInstance.StreamTask(context.Background(), "test task", func(chunk string) {}, func(toolCall llm.ToolCall) (string, error) {
		return "", nil
	})
	if err == nil {
		t.Error("Expected error for empty response, got nil")
	}

	if !strings.Contains(err.Error(), "빈 응답") {
		t.Errorf("Expected error message to contain '빈 응답', got '%v'", err)
	}
}

func TestStreamTask_WithToolCalls(t *testing.T) {
	mockProvider := llm.NewMockProvider()
	
	// First call: return tool call, second call: return empty (task complete)
	callCount := 0
	mockProvider.SetOnStreamChat(func(ctx context.Context, messages []llm.Message, tools []llm.Tool, onChunk func(string), onToolCall func(llm.ToolCall)) error {
		callCount++
		if callCount == 1 {
			// First iteration: return tool call
			onChunk("I'll execute")
			onToolCall(llm.ToolCall{
				ID:    "call_1",
				Name:  "execute_command",
				Input: map[string]interface{}{"command": "ls"},
			})
		} else {
			// Second iteration: no tool calls (task complete)
			onChunk("Task completed")
		}
		return nil
	})
	
	mockChatManager := chat.NewManager(mockProvider)
	mockFS := filesystem.NewMockFileSystem()
	mockSkillManager, _ := NewSkillManagerWithFS("/test/skills", mockFS)

	agentInstance := NewAgent(mockProvider, mockChatManager, mockSkillManager)

	var toolCalls []llm.ToolCall
	err := agentInstance.StreamTask(context.Background(), "test task", func(chunk string) {}, func(toolCall llm.ToolCall) (string, error) {
		toolCalls = append(toolCalls, toolCall)
		return "result", nil
	})

	if err != nil {
		t.Fatalf("StreamTask() failed: %v", err)
	}

	if len(toolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(toolCalls))
	}

	if len(toolCalls) > 0 && toolCalls[0].Name != "execute_command" {
		t.Errorf("Expected tool call name 'execute_command', got '%s'", toolCalls[0].Name)
	}
}
