package main

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func newSpinner() spinner.Model {
	spin := spinner.New()
	spin.Spinner = spinner.Spinner{
		Frames: []string{"-", "\\", "|", "/"},
		FPS:    120 * time.Millisecond,
	}
	spin.Style = rateLimitStyle
	return spin
}

func (m *tuiModel) adjustViewport() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	m.viewport.Width = max(10, m.width)
	contentHeight := m.height - m.input.Height() - 3
	if m.approval != nil {
		approval := renderApprovalPromptWithSelection(m.approval, m.width, m.approveIdx)
		contentHeight -= lineCount(approval)
	}
	m.viewport.Height = max(1, contentHeight)
}

func (m *tuiModel) refreshViewport() {
	m.viewport.SetContent(renderMessages(m.messages, m.viewport.Width))
	if m.followOutput {
		m.viewport.GotoBottom()
	}
}

func (m *tuiModel) handleViewportKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "pgup", "pgdown":
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		m.followOutput = m.viewport.AtBottom()
		return true, cmd
	case "home":
		m.viewport.GotoTop()
		m.followOutput = false
		return true, nil
	case "end":
		m.viewport.GotoBottom()
		m.followOutput = true
		return true, nil
	default:
		return false, nil
	}
}

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}
