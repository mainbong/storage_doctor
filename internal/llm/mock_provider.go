package llm

import (
	"context"
)

// MockProvider is a mock implementation of Provider for testing
type MockProvider struct {
	model           string
	chatResponse    string
	streamChunks    []string
	toolCalls       []ToolCall
	chatError       error
	streamError     error
	onStreamChat    func(ctx context.Context, messages []Message, tools []Tool, onChunk func(string), onToolCall func(ToolCall)) error
	onStreamChatWithTools func(ctx context.Context, messages []Message, tools []Tool, onChunk func(string), onToolCall func(ToolCall)) error
}

// NewMockProvider creates a new MockProvider instance
func NewMockProvider() *MockProvider {
	return &MockProvider{
		model:        "mock-model",
		streamChunks: make([]string, 0),
		toolCalls:     make([]ToolCall, 0),
	}
}

// SetChatResponse sets the response for Chat() calls
func (m *MockProvider) SetChatResponse(response string) {
	m.chatResponse = response
}

// SetStreamChunks sets the chunks for StreamChat() calls
func (m *MockProvider) SetStreamChunks(chunks []string) {
	m.streamChunks = chunks
}

// SetToolCalls sets the tool calls for StreamChatWithTools() calls
func (m *MockProvider) SetToolCalls(toolCalls []ToolCall) {
	m.toolCalls = toolCalls
}

// SetChatError sets an error to return for Chat() calls
func (m *MockProvider) SetChatError(err error) {
	m.chatError = err
}

// SetStreamError sets an error to return for StreamChat() calls
func (m *MockProvider) SetStreamError(err error) {
	m.streamError = err
}

// SetOnStreamChat sets a custom handler for StreamChat
func (m *MockProvider) SetOnStreamChat(handler func(ctx context.Context, messages []Message, tools []Tool, onChunk func(string), onToolCall func(ToolCall)) error) {
	m.onStreamChatWithTools = handler
}

func (m *MockProvider) StreamChat(ctx context.Context, messages []Message, onChunk func(string)) error {
	return m.StreamChatWithTools(ctx, messages, nil, onChunk, nil)
}

func (m *MockProvider) StreamChatWithTools(ctx context.Context, messages []Message, tools []Tool, onChunk func(string), onToolCall func(ToolCall)) error {
	if m.onStreamChatWithTools != nil {
		return m.onStreamChatWithTools(ctx, messages, tools, onChunk, onToolCall)
	}

	if m.streamError != nil {
		return m.streamError
	}

	// Send chunks
	for _, chunk := range m.streamChunks {
		if onChunk != nil {
			onChunk(chunk)
		}
	}

	// Send tool calls
	for _, toolCall := range m.toolCalls {
		if onToolCall != nil {
			onToolCall(toolCall)
		}
	}

	return nil
}

func (m *MockProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	if m.chatError != nil {
		return "", m.chatError
	}

	if m.chatResponse != "" {
		return m.chatResponse, nil
	}

	// If no response set, return concatenated stream chunks
	response := ""
	for _, chunk := range m.streamChunks {
		response += chunk
	}

	return response, nil
}

func (m *MockProvider) GetModel() string {
	return m.model
}





