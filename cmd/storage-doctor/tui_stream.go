package main

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mainbong/storage_doctor/internal/llm"
)

func (m *tuiModel) startStream(input string) tea.Cmd {
	m.streamCh = make(chan streamEvent, 32)
	go func() {
		err := agentInstance.StreamTask(context.Background(), input, func(chunk string) {
			m.streamCh <- streamEvent{chunk: chunk}
		}, func(toolCall llm.ToolCall) (string, error) {
			approved := true
			if needsApproval(toolCall) {
				resp := make(chan bool, 1)
				m.streamCh <- streamEvent{approval: &approvalRequest{tool: toolCall, response: resp}}
				approved = <-resp
			}
			if !approved {
				return "", fmt.Errorf("사용자가 실행을 취소했습니다")
			}
			result, err := executeToolCallForAgentApproved(context.Background(), toolCall, approved)
			msg := &chatMessage{
				role:    "tool",
				content: formatToolDisplay(toolCall, result, err),
			}
			if err != nil {
				msg.content = formatToolDisplay(toolCall, result, err)
			}
			m.streamCh <- streamEvent{sys: msg}
			return result, err
		})
		m.streamCh <- streamEvent{done: true, err: err}
		close(m.streamCh)
	}()
	return waitForStream(m.streamCh)
}

func waitForStream(ch <-chan streamEvent) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return streamEvent{done: true}
		}
		return msg
	}
}
