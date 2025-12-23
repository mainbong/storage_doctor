package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if m.approval != nil {
			return m.handleApprovalKey(msg)
		}
		if isSubmitKey(msg) && !m.streaming {
			value := strings.TrimSpace(m.input.Value())
			if value == "" {
				return m, nil
			}
			m.messages = append(m.messages, chatMessage{role: "user", content: value})
			m.messages = append(m.messages, chatMessage{role: "assistant", content: ""})
			m.streamIndex = len(m.messages) - 1
			m.streaming = true
			m.input.SetValue("")
			m.input.Blur()
			m.adjustInputHeight()
			return m, m.startStream(value)
		}

		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.adjustInputHeight()
		return m, cmd
	case streamEvent:
		return m.handleStreamEvent(msg)
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.adjustInputHeight()
	return m, cmd
}

func (m tuiModel) handleApprovalKey(msg tea.KeyMsg) (tuiModel, tea.Cmd) {
	switch msg.String() {
	case "y":
		m.approval.response <- true
		close(m.approval.response)
		m.approval = nil
		m.approveMax = 0
		m.input.Focus()
	case "n", "esc":
		m.approval.response <- false
		close(m.approval.response)
		m.approval = nil
		m.approveMax = 0
		m.messages = append(m.messages, chatMessage{
			role:    "system",
			content: "승인을 취소했습니다. 추가 요청을 입력해주세요.",
		})
		m.input.Focus()
	case "a":
		if m.approveMax == 3 {
			m.autoApprove[commandKeyFromApproval(m.approval)] = true
			m.approval.response <- true
			close(m.approval.response)
			m.approval = nil
			m.approveMax = 0
			m.input.Focus()
		}
	case "left", "up", "shift+tab":
		if m.approveMax > 0 {
			m.approveIdx = (m.approveIdx + m.approveMax - 1) % m.approveMax
		}
	case "right", "down", "tab":
		if m.approveMax > 0 {
			m.approveIdx = (m.approveIdx + 1) % m.approveMax
		}
	case "enter":
		if m.approveMax > 0 {
			approved := m.approveIdx != 1
			if m.approveMax == 3 && m.approveIdx == 2 {
				m.autoApprove[commandKeyFromApproval(m.approval)] = true
			}
			m.approval.response <- approved
			close(m.approval.response)
			m.approval = nil
			m.approveMax = 0
			if !approved {
				m.messages = append(m.messages, chatMessage{
					role:    "system",
					content: "승인을 취소했습니다. 추가 요청을 입력해주세요.",
				})
			}
			m.input.Focus()
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.approval.response <- false
			close(m.approval.response)
			m.approval = nil
			m.approveMax = 0
			m.messages = append(m.messages, chatMessage{
				role:    "system",
				content: "승인을 취소했습니다. 추가 요청을 입력해주세요.",
			})
			m.input.Focus()
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.adjustInputHeight()
			return m, cmd
		}
	}
	return m, nil
}

func (m tuiModel) handleStreamEvent(msg streamEvent) (tuiModel, tea.Cmd) {
	if msg.done {
		m.streaming = false
		m.streamIndex = -1
		if m.approval == nil {
			m.input.Focus()
		}
		if msg.err != nil {
			m.messages = append(m.messages, chatMessage{
				role:    "system",
				content: fmt.Sprintf("오류: %v", msg.err),
			})
		}
		return m, nil
	}
	if msg.sys != nil {
		m.messages = append(m.messages, *msg.sys)
		m.streamIndex = -1
	}
	if msg.approval != nil {
		if isAutoApproved(m.autoApprove, msg.approval.tool) {
			msg.approval.response <- true
			close(msg.approval.response)
		} else {
			m.approval = msg.approval
			m.approveIdx = 0
			m.approveMax = approvalOptionsCount(msg.approval.tool)
			m.input.Blur()
		}
	}
	if msg.chunk != "" {
		if m.streamIndex < 0 || m.streamIndex >= len(m.messages) || m.messages[m.streamIndex].role != "assistant" {
			m.messages = append(m.messages, chatMessage{role: "assistant", content: ""})
			m.streamIndex = len(m.messages) - 1
		}
		m.messages[m.streamIndex].content += msg.chunk
	}
	return m, waitForStream(m.streamCh)
}

func isSubmitKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "enter":
		return true
	default:
		return false
	}
}
