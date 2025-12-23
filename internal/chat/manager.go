package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mainbong/storage_doctor/internal/llm"
)

// Manager manages chat conversations and context
type Manager struct {
	provider           llm.Provider
	messages           []llm.Message
	maxMessages        int
	summarizeThreshold int
}

// NewManager creates a new chat manager
func NewManager(provider llm.Provider) *Manager {
	return &Manager{
		provider:           provider,
		messages:           make([]llm.Message, 0),
		maxMessages:        50,
		summarizeThreshold: 30,
	}
}

// AddMessage adds a message to the conversation
func (m *Manager) AddMessage(role, content string) {
	m.messages = append(m.messages, llm.Message{
		Role:    role,
		Content: content,
	})
}

// GetMessages returns all messages
func (m *Manager) GetMessages() []llm.Message {
	return m.messages
}

// StreamChat streams a chat response
func (m *Manager) StreamChat(ctx context.Context, userInput string, onChunk func(string)) error {
	return m.StreamChatWithTools(ctx, userInput, nil, onChunk, nil)
}

// StreamChatWithTools streams a chat response with tool support
func (m *Manager) StreamChatWithTools(ctx context.Context, userInput string, tools []llm.Tool, onChunk func(string), onToolCall func(llm.ToolCall)) error {
	// Add user message
	m.AddMessage("user", userInput)

	// Check if we need to summarize
	if len(m.messages) > m.summarizeThreshold {
		if err := m.summarizeContext(ctx); err != nil {
			// If summarization fails, continue with current messages
			fmt.Printf("Warning: failed to summarize context: %v\n", err)
		}
	}

	// Stream response with tools
	var assistantResponse strings.Builder
	var toolCalls []llm.ToolCall

	err := m.provider.StreamChatWithTools(ctx, m.messages, tools, func(chunk string) {
		assistantResponse.WriteString(chunk)
		onChunk(chunk)
	}, func(toolCall llm.ToolCall) {
		toolCalls = append(toolCalls, toolCall)
		if onToolCall != nil {
			onToolCall(toolCall)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to stream chat: %w", err)
	}

	// Add assistant response
	responseText := assistantResponse.String()
	if len(toolCalls) > 0 {
		// Format tool calls in response
		for _, tc := range toolCalls {
			responseText += fmt.Sprintf("\n<function_call>{\"name\":\"%s\",\"input\":%s}</function_call>", tc.Name, formatToolInput(tc.Input))
		}
	}
	m.AddMessage("assistant", responseText)

	return nil
}

// formatToolInput formats tool input as JSON string
func formatToolInput(input map[string]interface{}) string {
	data, err := json.Marshal(input)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// Chat sends a chat message and returns the full response
func (m *Manager) Chat(ctx context.Context, userInput string) (string, error) {
	var response strings.Builder
	err := m.StreamChat(ctx, userInput, func(chunk string) {
		response.WriteString(chunk)
	})
	return response.String(), err
}

// summarizeContext summarizes old messages to save tokens
func (m *Manager) summarizeContext(ctx context.Context) error {
	if len(m.messages) < 10 {
		return nil // Not enough messages to summarize
	}

	// Get old messages to summarize (keep last 5)
	oldMessages := m.messages[:len(m.messages)-5]
	recentMessages := m.messages[len(m.messages)-5:]

	// Create summary prompt
	summaryPrompt := "다음 대화 내용을 간결하게 요약해주세요. 중요한 정보(에러 메시지, 명령어, 파일 경로 등)는 반드시 포함해주세요:\n\n"
	for _, msg := range oldMessages {
		summaryPrompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	// Request summary
	summaryMessages := []llm.Message{
		{Role: "system", Content: "You are a helpful assistant that summarizes conversations while preserving important technical details."},
		{Role: "user", Content: summaryPrompt},
	}

	summary, err := m.provider.Chat(ctx, summaryMessages)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Replace old messages with summary
	m.messages = append([]llm.Message{
		{Role: "system", Content: "Previous conversation summary: " + summary},
	}, recentMessages...)

	return nil
}

// Clear clears all messages
func (m *Manager) Clear() {
	m.messages = make([]llm.Message, 0)
}

// SetSystemPrompt sets the system prompt
func (m *Manager) SetSystemPrompt(prompt string) {
	// Remove existing system prompt if any
	filtered := make([]llm.Message, 0, len(m.messages))
	for _, msg := range m.messages {
		if msg.Role != "system" {
			filtered = append(filtered, msg)
		}
	}
	m.messages = filtered

	// Add new system prompt at the beginning
	m.messages = append([]llm.Message{
		{Role: "system", Content: prompt},
	}, m.messages...)
}
