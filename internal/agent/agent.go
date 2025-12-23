package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/mainbong/storage_doctor/internal/chat"
	"github.com/mainbong/storage_doctor/internal/llm"
)

// Agent represents an autonomous agent that can use tools
type Agent struct {
	llmProvider   llm.Provider
	chatManager   *chat.Manager
	skillManager  *SkillManager
	tools         []llm.Tool
	maxIterations int
}

// NewAgent creates a new agent
func NewAgent(llmProvider llm.Provider, chatManager *chat.Manager, skillManager *SkillManager) *Agent {
	return &Agent{
		llmProvider:   llmProvider,
		chatManager:   chatManager,
		skillManager:  skillManager,
		tools:         llm.GetTools(),
		maxIterations: 10, // Maximum tool calls per task
	}
}

// ExecuteTask executes a task autonomously using tools
func (a *Agent) ExecuteTask(ctx context.Context, task string, onToolCall func(llm.ToolCall) (string, error)) (string, error) {
	// Build system prompt with skill metadata
	systemPrompt := a.buildSystemPrompt()
	a.chatManager.SetSystemPrompt(systemPrompt)

	// Add user task
	a.chatManager.AddMessage("user", task)

	iterations := 0
	var finalResponse strings.Builder

	for iterations < a.maxIterations {
		iterations++

		// Get current messages
		messages := a.chatManager.GetMessages()

		// Stream response and collect tool calls
		var responseText strings.Builder
		var toolCalls []llm.ToolCall

		err := a.llmProvider.StreamChatWithTools(ctx, messages, a.tools, func(chunk string) {
			responseText.WriteString(chunk)
		}, func(toolCall llm.ToolCall) {
			toolCalls = append(toolCalls, toolCall)
		})

		if err != nil {
			return "", fmt.Errorf("LLM 호출 실패: %w", err)
		}

		response := responseText.String()
		finalResponse.WriteString(response)
		finalResponse.WriteString("\n\n")

		// If no tool calls, task is complete
		if len(toolCalls) == 0 {
			// Add final response to chat
			a.chatManager.AddMessage("assistant", response)
			break
		}

		// Execute tool calls
		var toolResults []string
		for _, toolCall := range toolCalls {
			result, err := onToolCall(toolCall)
			if err != nil {
				toolResults = append(toolResults, chat.FormatToolCall(toolCall.Name, fmt.Sprintf("오류: %v", err), false))
			} else {
				toolResults = append(toolResults, chat.FormatToolCall(toolCall.Name, result, true))
			}
		}

		// Add tool results to conversation
		toolResultsText := strings.Join(toolResults, "\n\n")
		a.chatManager.AddMessage("user", toolResultsText)

		// Check if task is complete (no more tool calls needed)
		// This will be determined in the next iteration
	}

	// Add final response
	a.chatManager.AddMessage("assistant", finalResponse.String())

	return finalResponse.String(), nil
}

// buildSystemPrompt builds the system prompt with skill metadata
func (a *Agent) buildSystemPrompt() string {
	var builder strings.Builder

	builder.WriteString(`당신은 스토리지 문제 해결을 전문으로 하는 자율적인 AI Agent입니다.

주요 역할:
1. 사용자가 설명한 스토리지 문제를 분석하고 진단
2. 필요한 도구를 자율적으로 선택하고 실행
3. 여러 단계의 작업을 계획하고 순차적으로 수행
4. 중간 결과를 분석하고 다음 단계를 결정
5. 작업이 완료될 때까지 반복적으로 도구를 사용

작업 방식:
- 문제를 분석하고 해결 계획을 수립
- 필요한 도구를 순차적으로 실행
- 각 도구 실행 결과를 분석
- 결과에 따라 다음 단계 결정
- 목표 달성 시 작업 종료

중요 규칙:
- 명령어 실행 전 사용자 승인 필요 (auto_approve가 설정되지 않은 경우)
- 파일 수정 전 자동 백업 생성
- 위험한 작업은 사용자에게 확인
- 한국어로 응답

`)

	// Add skill metadata
	skillMetadata := a.skillManager.GetSkillMetadata()
	if skillMetadata != "" {
		builder.WriteString(skillMetadata)
		builder.WriteString("\n")
	}

	// Add tool descriptions
	builder.WriteString("사용 가능한 도구:\n")
	for _, tool := range a.tools {
		builder.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
	}

	return builder.String()
}

// ActivateSkill activates a skill and adds it to the context
func (a *Agent) ActivateSkill(skillName string) error {
	skillContent, err := a.skillManager.ActivateSkill(skillName)
	if err != nil {
		return fmt.Errorf("스킬 활성화 실패: %w", err)
	}

	// Add skill content to system prompt or as a message
	currentPrompt := a.chatManager.GetMessages()
	if len(currentPrompt) > 0 && currentPrompt[0].Role == "system" {
		// Append skill to system prompt
		newPrompt := currentPrompt[0].Content + "\n\n" + skillContent
		a.chatManager.SetSystemPrompt(newPrompt)
	} else {
		// Add as system message
		a.chatManager.AddMessage("system", skillContent)
	}

	return nil
}

// StreamTask executes a task with streaming response
func (a *Agent) StreamTask(ctx context.Context, task string, onChunk func(string), onToolCall func(llm.ToolCall) (string, error)) error {
	// Build system prompt
	systemPrompt := a.buildSystemPrompt()
	a.chatManager.SetSystemPrompt(systemPrompt)

	// Add user task
	a.chatManager.AddMessage("user", task)

	iterations := 0

	for iterations < a.maxIterations {
		iterations++

		// Get current messages
		messages := a.chatManager.GetMessages()

		// Stream response and collect tool calls
		var responseText strings.Builder
		var toolCalls []llm.ToolCall

		err := a.llmProvider.StreamChatWithTools(ctx, messages, a.tools, func(chunk string) {
			onChunk(chunk)
			responseText.WriteString(chunk)
		}, func(toolCall llm.ToolCall) {
			toolCalls = append(toolCalls, toolCall)
		})

		if err != nil {
			return fmt.Errorf("LLM 호출 실패: %w", err)
		}

		response := responseText.String()

		// Check if we got any response at all
		if response == "" && len(toolCalls) == 0 {
			return fmt.Errorf("LLM에서 응답을 받지 못했습니다 (빈 응답)")
		}

		// If no tool calls, task is complete
		if len(toolCalls) == 0 {
			a.chatManager.AddMessage("assistant", response)
			return nil
		}

		// Execute tool calls
		var toolResults []string
		for _, toolCall := range toolCalls {
			result, err := onToolCall(toolCall)
			if err != nil {
				toolResults = append(toolResults, chat.FormatToolCall(toolCall.Name, fmt.Sprintf("오류: %v", err), false))
			} else {
				toolResults = append(toolResults, chat.FormatToolCall(toolCall.Name, result, true))
			}
		}

		// Add tool results to conversation
		toolResultsText := strings.Join(toolResults, "\n\n")
		a.chatManager.AddMessage("user", toolResultsText)

		// Continue to next iteration
		onChunk("\n\n[도구 실행 완료, 다음 단계 진행 중...]\n\n")
	}

	return nil
}
