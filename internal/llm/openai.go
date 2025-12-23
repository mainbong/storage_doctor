package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mainbong/storage_doctor/internal/config"
	"github.com/mainbong/storage_doctor/internal/httpclient"
)

type OpenAIProvider struct {
	apiKey string
	model  string
	client httpclient.HTTPClient
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(cfg *config.Config) *OpenAIProvider {
	return NewOpenAIProviderWithClient(cfg, httpclient.NewDefaultHTTPClient())
}

// NewOpenAIProviderWithClient creates a new OpenAI provider with a custom HTTPClient (for testing)
func NewOpenAIProviderWithClient(cfg *config.Config, client httpclient.HTTPClient) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey: cfg.OpenAI.APIKey,
		model:  cfg.OpenAI.Model,
		client: client,
	}
}

type openaiRequest struct {
	Model     string          `json:"model"`
	Messages  []openaiMessage `json:"messages"`
	Stream    bool            `json:"stream"`
	MaxTokens int             `json:"max_tokens,omitempty"`
	Tools     []openaiTool    `json:"tools,omitempty"`
}

type openaiMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []openaiContentBlock
}

type openaiContentBlock struct {
	Type     string          `json:"type"` // "text" or "tool_call"
	Text     string          `json:"text,omitempty"`
	ToolCall *openaiToolCall `json:"tool_call,omitempty"`
}

type openaiToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function openaiFunction `json:"function"`
}

type openaiFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

type openaiTool struct {
	Type     string             `json:"type"`
	Function openaiToolFunction `json:"function"`
}

type openaiToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type openaiStreamResponse struct {
	Choices []openaiChoice `json:"choices"`
}

type openaiChoice struct {
	Delta openaiDelta `json:"delta"`
}

type openaiDelta struct {
	Content   string                `json:"content"`
	ToolCalls []openaiToolCallDelta `json:"tool_calls,omitempty"`
}

type openaiToolCallDelta struct {
	Index    int                 `json:"index"`
	ID       string              `json:"id,omitempty"`
	Type     string              `json:"type,omitempty"`
	Function openaiFunctionDelta `json:"function,omitempty"`
}

type openaiFunctionDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

func (p *OpenAIProvider) StreamChat(ctx context.Context, messages []Message, onChunk func(string)) error {
	return p.StreamChatWithTools(ctx, messages, nil, onChunk, nil)
}

func (p *OpenAIProvider) StreamChatWithTools(ctx context.Context, messages []Message, tools []Tool, onChunk func(string), onToolCall func(ToolCall)) error {
	if p.apiKey == "" {
		return fmt.Errorf("openai API key not set")
	}

	// Convert messages to OpenAI format
	openaiMessages := make([]openaiMessage, 0, len(messages))
	for _, msg := range messages {
		// Check if message contains tool results
		if strings.Contains(msg.Content, "<tool_result") {
			// For tool results, keep as text
			openaiMessages = append(openaiMessages, openaiMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		} else {
			openaiMessages = append(openaiMessages, openaiMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	reqBody := openaiRequest{
		Model:     p.model,
		Messages:  openaiMessages,
		Stream:    true,
		MaxTokens: 4096,
	}

	// Add tools if provided
	if tools != nil && len(tools) > 0 {
		openaiTools := make([]openaiTool, 0, len(tools))
		for _, tool := range tools {
			openaiTools = append(openaiTools, openaiTool{
				Type: "function",
				Function: openaiToolFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.InputSchema,
				},
			})
		}
		reqBody.Tools = openaiTools
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai API error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var currentToolCalls map[int]*ToolCall = make(map[int]*ToolCall)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse SSE format: "data: {...}"
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var streamResp openaiStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) > 0 {
				delta := streamResp.Choices[0].Delta

				// Handle text content
				if delta.Content != "" {
					onChunk(delta.Content)
				}

				// Handle tool calls
				if len(delta.ToolCalls) > 0 && onToolCall != nil {
					for _, toolCallDelta := range delta.ToolCalls {
						idx := toolCallDelta.Index
						if currentToolCalls[idx] == nil {
							currentToolCalls[idx] = &ToolCall{
								ID:    toolCallDelta.ID,
								Name:  toolCallDelta.Function.Name,
								Input: make(map[string]interface{}),
							}
						}

						if toolCallDelta.Function.Name != "" {
							currentToolCalls[idx].Name = toolCallDelta.Function.Name
						}
						if toolCallDelta.ID != "" && currentToolCalls[idx].ID == "" {
							currentToolCalls[idx].ID = toolCallDelta.ID
						}

						if toolCallDelta.Function.Arguments != "" {
							// Accumulate arguments
							existingArgs := ""
							if args, ok := currentToolCalls[idx].Input["_raw_args"].(string); ok {
								existingArgs = args
							}
							existingArgs += toolCallDelta.Function.Arguments
							currentToolCalls[idx].Input["_raw_args"] = existingArgs

							// Try to parse as JSON
							var parsedArgs map[string]interface{}
							if err := json.Unmarshal([]byte(existingArgs), &parsedArgs); err == nil {
								currentToolCalls[idx].Input = parsedArgs
								delete(currentToolCalls[idx].Input, "_raw_args")
							}
						}

					}
				}
			}
		}
	}

	// Process any remaining tool calls
	if onToolCall != nil {
		for _, toolCall := range currentToolCalls {
			if toolCall.ID != "" && toolCall.Name != "" {
				onToolCall(*toolCall)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read stream: %w", err)
	}

	return nil
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	var fullResponse strings.Builder

	err := p.StreamChat(ctx, messages, func(chunk string) {
		fullResponse.WriteString(chunk)
	})

	if err != nil {
		return "", err
	}

	return fullResponse.String(), nil
}

func (p *OpenAIProvider) GetModel() string {
	return p.model
}
