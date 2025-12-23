package chat

import (
	"strings"
	"testing"
)

func TestParseResponse(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		expectedActions int
		expectedText   string
	}{
		{
			name:           "Single function call",
			text:           `<function_call>{"name":"execute_command","input":{"command":"ls"}}</function_call>`,
			expectedActions: 1,
			expectedText:   "",
		},
		{
			name:           "Multiple function calls",
			text:           `<function_call>{"name":"read_file","input":{"path":"/test"}}</function_call><function_call>{"name":"write_file","input":{"path":"/test2","content":"data"}}</function_call>`,
			expectedActions: 2,
			expectedText:   "",
		},
		{
			name:           "No function calls",
			text:           "Just regular text",
			expectedActions: 0,
			expectedText:   "Just regular text",
		},
		{
			name:           "Function call with text",
			text:           "Here's the result: <function_call>{\"name\":\"execute_command\",\"input\":{\"command\":\"ls\"}}</function_call>",
			expectedActions: 1,
			expectedText:   "Here's the result:",
		},
		{
			name:           "Invalid JSON",
			text:           `<function_call>{invalid json}</function_call>`,
			expectedActions: 0,
			expectedText:   `<function_call>{invalid json}</function_call>`,
		},
		{
			name:           "Command marker",
			text:           "[COMMAND: kubectl get pods]",
			expectedActions: 1,
			expectedText:   "",
		},
		{
			name:           "Read file marker",
			text:           "[READ_FILE: /path/to/file]",
			expectedActions: 1,
			expectedText:   "",
		},
		{
			name:           "Search marker",
			text:           "[SEARCH: kubernetes pv issues]",
			expectedActions: 1,
			expectedText:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions, text := ParseResponse(tt.text)
			if len(actions) != tt.expectedActions {
				t.Errorf("Expected %d actions, got %d", tt.expectedActions, len(actions))
			}
			if strings.TrimSpace(text) != strings.TrimSpace(tt.expectedText) {
				t.Errorf("Expected text '%s', got '%s'", tt.expectedText, text)
			}
		})
	}
}

func TestFormatToolCall(t *testing.T) {
	result := FormatToolCall("test_tool", "test result", true)
	if result == "" {
		t.Error("Expected formatted tool call, got empty string")
	}

	if !strings.Contains(result, "test_tool") {
		t.Error("Expected formatted result to contain tool name")
	}

	if !strings.Contains(result, "test result") {
		t.Error("Expected formatted result to contain result")
	}
}

func TestFormatToolCall_Error(t *testing.T) {
	result := FormatToolCall("test_tool", "error message", false)
	if result == "" {
		t.Error("Expected formatted tool call, got empty string")
	}

	if !strings.Contains(result, "실패") {
		t.Error("Expected formatted result to contain '실패'")
	}

	if !strings.Contains(result, "error message") {
		t.Error("Expected formatted result to contain error message")
	}
}

