package main

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mainbong/storage_doctor/internal/llm"
)

type streamEvent struct {
	chunk    string
	done     bool
	err      error
	sys      *chatMessage
	approval *approvalRequest
	rate     *rateLimitStatus
}

type chatMessage struct {
	role    string
	content string
}

type approvalRequest struct {
	tool     llm.ToolCall
	response chan bool
}

type rateLimitStatus struct {
	waiting bool
	wait    time.Duration
}

type tuiModel struct {
	input        textarea.Model
	messages     []chatMessage
	streaming    bool
	streamIndex  int
	streamCh     chan streamEvent
	approval     *approvalRequest
	approveIdx   int
	approveMax   int
	autoApprove  map[string]bool
	viewport     viewport.Model
	followOutput bool
	spinner      spinner.Model
	rateLimit    *rateLimitStatus
	width        int
	height       int
}

func runTUI() error {
	model := newTUIModel()
	program := tea.NewProgram(model, tea.WithMouseCellMotion())
	_, err := program.Run()
	return err
}

func newTUIModel() tuiModel {
	input := textarea.New()
	input.Prompt = "> "
	input.Placeholder = "type here..."
	input.ShowLineNumbers = false
	input.CharLimit = 0
	input.Focus()
	input.KeyMap.InsertNewline.SetKeys("shift+enter")
	input.FocusedStyle = textarea.Style{
		Prompt:      promptStyle,
		Text:        inputTextStyle,
		Placeholder: placeholderStyle,
		CursorLine:  cursorLineStyle,
	}
	input.BlurredStyle = textarea.Style{
		Prompt:      promptStyle,
		Text:        inputTextStyle,
		Placeholder: placeholderStyle,
	}

	return tuiModel{
		input:        input,
		streamIndex:  -1,
		autoApprove:  make(map[string]bool),
		viewport:     viewport.New(0, 0),
		followOutput: true,
		spinner:      newSpinner(),
	}
}

func (m tuiModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m *tuiModel) resize() {
	m.input.SetWidth(max(10, m.width-4))
	m.adjustInputHeight()
	m.adjustViewport()
}

func (m *tuiModel) adjustInputHeight() {
	maxHeight := min(6, max(1, m.height/6))
	lines := 1
	if value := m.input.Value(); value != "" {
		lines = strings.Count(value, "\n") + 1
	}
	m.input.SetHeight(min(maxHeight, max(1, lines)))
	m.adjustViewport()
}
