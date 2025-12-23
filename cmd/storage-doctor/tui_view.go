package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m tuiModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	divider := strings.Repeat("-", m.width)
	hintText := "? 단축키 안내 (추가 예정) | Enter 전송 | Shift+Enter 줄바꿈"
	hint := lipgloss.PlaceHorizontal(m.width, lipgloss.Left, hintStyle.Render(hintText))
	if m.streaming {
		hint = lipgloss.PlaceHorizontal(m.width, lipgloss.Left, hintStyle.Render("응답 생성 중... (Ctrl+C 종료)"))
	}

	content := renderMessages(m.messages, m.width)
	approval := ""
	if m.approval != nil {
		approval = renderApprovalPromptWithSelection(m.approval, m.width, m.approveIdx)
	}

	parts := []string{}
	if content != "" {
		parts = append(parts, content)
	}
	if approval != "" {
		parts = append(parts, approval)
	}
	parts = append(parts, divider, m.input.View(), divider, hint)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

var _ tea.Model = tuiModel{}
