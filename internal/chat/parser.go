package chat

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ParsedAction represents a parsed action from LLM response
type ParsedAction struct {
	Type        string                 `json:"type"`
	ToolName    string                 `json:"tool_name,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
	Description string                 `json:"description,omitempty"`
}

// ParseResponse parses LLM response to extract function calls and actions
func ParseResponse(response string) ([]ParsedAction, string) {
	var actions []ParsedAction
	var textResponse strings.Builder

	// Pattern 1: JSON function call blocks
	// Format: <function_call>{"name": "execute_command", "input": {...}}</function_call>
	jsonBlockPattern := regexp.MustCompile(`<function_call>(.*?)</function_call>`)
	matches := jsonBlockPattern.FindAllStringSubmatch(response, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		var toolCall struct {
			Name        string                 `json:"name"`
			Input       map[string]interface{} `json:"input"`
			Description string                 `json:"description,omitempty"`
		}

		if err := json.Unmarshal([]byte(match[1]), &toolCall); err == nil {
			actions = append(actions, ParsedAction{
				Type:        "function_call",
				ToolName:    toolCall.Name,
				Parameters:  toolCall.Input,
				Description: toolCall.Description,
			})
			// Remove function call from text response
			response = strings.Replace(response, match[0], "", 1)
		}
	}

	// Pattern 2: Simple command markers
	// Format: [COMMAND: kubectl get pods]
	commandPattern := regexp.MustCompile(`\[COMMAND:\s*(.+?)\]`)
	commandMatches := commandPattern.FindAllStringSubmatch(response, -1)

	for _, match := range commandMatches {
		if len(match) < 2 {
			continue
		}
		actions = append(actions, ParsedAction{
			Type:     "function_call",
			ToolName: "execute_command",
			Parameters: map[string]interface{}{
				"command":     strings.TrimSpace(match[1]),
				"description": "LLM이 제안한 명령어",
			},
		})
		response = strings.Replace(response, match[0], "", 1)
	}

	// Pattern 3: File operations
	// Format: [READ_FILE: /path/to/file] or [WRITE_FILE: /path/to/file]
	readFilePattern := regexp.MustCompile(`\[READ_FILE:\s*(.+?)\]`)
	readMatches := readFilePattern.FindAllStringSubmatch(response, -1)
	for _, match := range readMatches {
		if len(match) < 2 {
			continue
		}
		actions = append(actions, ParsedAction{
			Type:     "function_call",
			ToolName: "read_file",
			Parameters: map[string]interface{}{
				"path": strings.TrimSpace(match[1]),
			},
		})
		response = strings.Replace(response, match[0], "", 1)
	}

	writeFilePattern := regexp.MustCompile(`\[WRITE_FILE:\s*(.+?)\]\s*` + "```" + `(?:.*?)\n(.*?)` + "```")
	writeMatches := writeFilePattern.FindAllStringSubmatch(response, -1)
	for _, match := range writeMatches {
		if len(match) < 3 {
			continue
		}
		actions = append(actions, ParsedAction{
			Type:     "function_call",
			ToolName: "write_file",
			Parameters: map[string]interface{}{
				"path":    strings.TrimSpace(match[1]),
				"content": strings.TrimSpace(match[2]),
			},
		})
		response = strings.Replace(response, match[0], "", 1)
	}

	// Pattern 4: Search requests
	// Format: [SEARCH: query]
	searchPattern := regexp.MustCompile(`\[SEARCH:\s*(.+?)\]`)
	searchMatches := searchPattern.FindAllStringSubmatch(response, -1)
	for _, match := range searchMatches {
		if len(match) < 2 {
			continue
		}
		actions = append(actions, ParsedAction{
			Type:     "function_call",
			ToolName: "search_web",
			Parameters: map[string]interface{}{
				"query": strings.TrimSpace(match[1]),
			},
		})
		response = strings.Replace(response, match[0], "", 1)
	}

	// Clean up response text
	textResponse.WriteString(strings.TrimSpace(response))

	return actions, textResponse.String()
}

// FormatToolCall formats a tool call for LLM context
func FormatToolCall(toolName string, result string, success bool) string {
	status := "성공"
	if !success {
		status = "실패"
	}
	return fmt.Sprintf("<tool_result name=\"%s\" status=\"%s\">%s</tool_result>", toolName, status, result)
}
