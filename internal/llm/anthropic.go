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
	"time"

	"github.com/mainbong/storage_doctor/internal/config"
	"github.com/mainbong/storage_doctor/internal/httpclient"
	"github.com/mainbong/storage_doctor/internal/logger"
)

type AnthropicProvider struct {
	apiKey  string
	model   string
	client  httpclient.HTTPClient
	limiter *RateLimiter
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(cfg *config.Config) *AnthropicProvider {
	return NewAnthropicProviderWithClient(cfg, httpclient.NewDefaultHTTPClient())
}

// NewAnthropicProviderWithClient creates a new Anthropic provider with a custom HTTPClient (for testing)
func NewAnthropicProviderWithClient(cfg *config.Config, client httpclient.HTTPClient) *AnthropicProvider {
	tokensPerMinute, requestsPerMinute := defaultRateLimits("anthropic")
	return &AnthropicProvider{
		apiKey:  cfg.Anthropic.APIKey,
		model:   cfg.Anthropic.Model,
		client:  client,
		limiter: NewRateLimiter(time.Minute, tokensPerMinute, requestsPerMinute),
	}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
	Stream    bool               `json:"stream"`
	System    string             `json:"system,omitempty"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicContentBlock struct {
	Type      string                 `json:"type"` // "text" or "tool_use"
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`          // Tool use ID (in content_block)
	ToolUseID string                 `json:"tool_use_id,omitempty"` // Legacy field
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
}

type anthropicTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type anthropicStreamResponse struct {
	Type         string                 `json:"type"`
	Delta        *anthropicDelta        `json:"delta,omitempty"`
	ContentBlock *anthropicContentBlock `json:"content_block,omitempty"`
	Message      *anthropicMessage      `json:"message,omitempty"`
}

type anthropicDelta struct {
	Type        string                 `json:"type"` // "text_delta" or "input_json_delta"
	Text        string                 `json:"text,omitempty"`
	PartialJSON string                 `json:"partial_json,omitempty"` // For input_json_delta
	ToolUseID   string                 `json:"tool_use_id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Input       map[string]interface{} `json:"input,omitempty"`
}

func (p *AnthropicProvider) StreamChat(ctx context.Context, messages []Message, onChunk func(string)) error {
	return p.StreamChatWithTools(ctx, messages, nil, onChunk, nil)
}

// StreamChatWithTools streams chat with tool support
func (p *AnthropicProvider) StreamChatWithTools(ctx context.Context, messages []Message, tools []Tool, onChunk func(string), onToolCall func(ToolCall)) error {
	if p.apiKey == "" {
		return fmt.Errorf("anthropic API key not set")
	}
	if err := p.limiter.Wait(ctx, EstimateTokens(messages)); err != nil {
		return fmt.Errorf("rate limit 대기 실패: %w", err)
	}

	// Convert messages to Anthropic format
	anthropicMessages := make([]anthropicMessage, 0, len(messages))
	var systemPrompt string

	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			continue
		}

		// Skip empty messages
		if strings.TrimSpace(msg.Content) == "" {
			continue
		}

		// Regular text message - Anthropic requires content array with text blocks
		content := []anthropicContentBlock{
			{
				Type: "text",
				Text: msg.Content,
			},
		}

		anthropicMessages = append(anthropicMessages, anthropicMessage{
			Role:    msg.Role,
			Content: content,
		})
	}

	// Ensure at least one message exists
	if len(anthropicMessages) == 0 {
		return fmt.Errorf("no valid messages to send")
	}

	// Debug: Log request details
	logger.Debug("Anthropic API 호출: Model=%s, Messages=%d, Tools=%d", p.model, len(anthropicMessages), len(tools))

	reqBody := anthropicRequest{
		Model:     p.model,
		MaxTokens: 4096,
		Messages:  anthropicMessages,
		Stream:    true,
	}
	if systemPrompt != "" {
		reqBody.System = systemPrompt
	}

	// Add tools if provided
	if tools != nil && len(tools) > 0 {
		anthropicTools := make([]anthropicTool, 0, len(tools))
		for _, tool := range tools {
			anthropicTools = append(anthropicTools, anthropicTool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.InputSchema,
			})
		}
		reqBody.Tools = anthropicTools
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		errorMsg := fmt.Sprintf("HTTP 요청 실패: %v", err)
		logger.Error("%s", errorMsg)
		return fmt.Errorf("HTTP 요청 실패: %w", err)
	}
	defer resp.Body.Close()
	updateLimiterFromHeaders(p.limiter, resp.Header, "anthropic-ratelimit-limit-tokens", "anthropic-ratelimit-limit-requests")

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errorMsg := fmt.Sprintf("anthropic API error: HTTP %d\n응답 본문: %s", resp.StatusCode, string(body))
		// Log detailed error for debugging
		logger.Error("%s", errorMsg)
		return fmt.Errorf("anthropic API error: HTTP %d - %s", resp.StatusCode, string(body))
	}

	logger.Debug("Anthropic API 응답 수신: HTTP %d", resp.StatusCode)

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	type toolBuffer struct {
		id        string
		name      string
		input     map[string]interface{}
		inputJSON strings.Builder
	}

	var currentTool *toolBuffer
	var receivedChunks int
	var receivedToolCalls int

	finalizeTool := func(reason string) {
		if currentTool == nil {
			return
		}
		if currentTool.id == "" || currentTool.name == "" || onToolCall == nil {
			logger.Warn("Tool call 완료 실패 (%s): ToolUseID=%s, ToolName=%s, onToolCall=%v",
				reason, currentTool.id, currentTool.name, onToolCall != nil)
			currentTool = nil
			return
		}

		if currentTool.input == nil {
			currentTool.input = make(map[string]interface{})
		}

		if currentTool.inputJSON.Len() > 0 {
			inputStr := currentTool.inputJSON.String()
			logger.Debug("Tool Input JSON 파싱 시도: %s", inputStr)
			if err := json.Unmarshal([]byte(inputStr), &currentTool.input); err != nil {
				logger.Warn("Tool Input JSON 파싱 실패: %v", err)
			} else {
				logger.Debug("Tool Input 파싱 성공")
			}
		}

		receivedToolCalls++
		logger.Debug("Tool call 완료: %s (ID: %s)", currentTool.name, currentTool.id)
		onToolCall(ToolCall{
			ID:    currentTool.id,
			Name:  currentTool.name,
			Input: currentTool.input,
		})
		currentTool = nil
	}

loop:
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse SSE format: "data: {...}"
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				logger.Debug("스트림 완료: Chunks=%d, ToolCalls=%d", receivedChunks, receivedToolCalls)
				break
			}

			// Log raw data for debugging (first few events only to avoid spam)
			if receivedChunks == 0 && receivedToolCalls == 0 && len(data) < 500 {
				logger.Debug("원시 스트림 데이터: %s", data)
			}

			var streamResp anthropicStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				logger.Warn("JSON 파싱 실패: %v, 데이터: %s", err, data)
				// Skip invalid JSON
				continue
			}

			// Debug: Log all event types (only non-text deltas to reduce noise)
			if streamResp.Type != "content_block_delta" || streamResp.Delta == nil || streamResp.Delta.Type != "text_delta" {
				logger.Debug("스트림 이벤트: Type=%s", streamResp.Type)
			}

			switch streamResp.Type {
			case "content_block_start":
				if streamResp.ContentBlock != nil {
					if streamResp.ContentBlock.Type == "tool_use" {
						if currentTool != nil {
							finalizeTool("content_block_start")
						}
						// ID can be in either "id" or "tool_use_id" field
						currentTool = &toolBuffer{
							id:    streamResp.ContentBlock.ID,
							name:  streamResp.ContentBlock.Name,
							input: streamResp.ContentBlock.Input,
						}
						if currentTool.id == "" {
							currentTool.id = streamResp.ContentBlock.ToolUseID
						}
						logger.Debug("Tool use 시작: %s (ID: %s)", currentTool.name, currentTool.id)
						if currentTool.id == "" {
							logger.Warn("Tool use 시작 시 ID가 비어있음!")
						}
						if currentTool.name == "" {
							logger.Warn("Tool use 시작 시 Name이 비어있음!")
						}
					} else if streamResp.ContentBlock.Type == "text" {
						logger.Debug("Text content block 시작")
					} else {
						logger.Debug("content_block_start: Type=%s", streamResp.ContentBlock.Type)
					}
				} else {
					logger.Warn("content_block_start에 ContentBlock이 없음")
				}
			case "content_block_delta":
				if streamResp.Delta != nil {
					if streamResp.Delta.Type == "text_delta" && streamResp.Delta.Text != "" {
						receivedChunks++
						onChunk(streamResp.Delta.Text)
					} else if streamResp.Delta.Type == "input_json_delta" {
						// Anthropic sends tool input as incremental JSON strings in partial_json
						if streamResp.Delta.PartialJSON != "" {
							if currentTool == nil {
								logger.Warn("Tool Input JSON 수신했지만 Tool use가 시작되지 않음")
								continue
							}
							currentTool.inputJSON.WriteString(streamResp.Delta.PartialJSON)
							logger.Debug("Tool Input JSON 누적 중... (현재 길이: %d)", currentTool.inputJSON.Len())
						}
					} else if streamResp.Delta.Type == "tool_use" {
						// Legacy format (shouldn't happen with new API)
						if currentTool == nil {
							currentTool = &toolBuffer{
								input: make(map[string]interface{}),
							}
						}
						if currentTool.input == nil {
							currentTool.input = make(map[string]interface{})
						}
						if streamResp.Delta.ToolUseID != "" && currentTool.id == "" {
							currentTool.id = streamResp.Delta.ToolUseID
							logger.Debug("Tool Use ID 설정: %s", currentTool.id)
						}
						if streamResp.Delta.Name != "" && currentTool.name == "" {
							currentTool.name = streamResp.Delta.Name
							logger.Debug("Tool Name 설정: %s", currentTool.name)
						}
						if streamResp.Delta.Input != nil {
							// Accumulate input JSON
							for k, v := range streamResp.Delta.Input {
								if str, ok := v.(string); ok {
									currentTool.inputJSON.WriteString(str)
								} else {
									currentTool.input[k] = v
								}
							}
							logger.Debug("Tool Input 누적 중... (현재 길이: %d)", currentTool.inputJSON.Len())
						}
					} else {
						logger.Debug("content_block_delta: Type=%s, Text=%s, PartialJSON=%s",
							streamResp.Delta.Type, streamResp.Delta.Text, streamResp.Delta.PartialJSON)
					}
				} else {
					logger.Warn("content_block_delta에 Delta가 없음")
				}
			case "content_block_stop":
				if currentTool != nil {
					logger.Debug("content_block_stop: ToolUseID=%s, ToolName=%s, InputLen=%d",
						currentTool.id, currentTool.name, currentTool.inputJSON.Len())
				}
				finalizeTool("content_block_stop")
			case "message_stop":
				logger.Debug("메시지 스트림 종료: Chunks=%d, ToolCalls=%d", receivedChunks, receivedToolCalls)
				finalizeTool("message_stop")
				break loop
			case "error":
				// Handle error events from stream
				errorData, _ := json.Marshal(streamResp)
				logger.Error("스트림 에러 이벤트: %s", string(errorData))
				return fmt.Errorf("스트림에서 에러 이벤트 수신")
			}
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		errorMsg := fmt.Sprintf("스트림 읽기 실패: %v", err)
		logger.Error("%s", errorMsg)
		return fmt.Errorf("스트림 읽기 실패: %w", err)
	}

	// If we received no chunks and no tool calls, that's suspicious
	if receivedChunks == 0 && receivedToolCalls == 0 {
		logger.Warn("스트림에서 데이터를 받지 못했습니다 (Chunks=0, ToolCalls=0)")
	}

	return nil
}

func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	var fullResponse strings.Builder

	err := p.StreamChat(ctx, messages, func(chunk string) {
		fullResponse.WriteString(chunk)
	})

	if err != nil {
		return "", err
	}

	return fullResponse.String(), nil
}

func (p *AnthropicProvider) GetModel() string {
	return p.model
}
