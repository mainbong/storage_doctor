package terminal

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"
)

// Renderer formats markdown-ish output for TTY displays.
type Renderer struct {
	writer     io.Writer
	buffer     strings.Builder
	inCode     bool
	tableLines []string
	codeLang   string
	linePrefix string
}

// NewRenderer creates a renderer that writes to the provided writer.
func NewRenderer(writer io.Writer) *Renderer {
	return &Renderer{writer: writer}
}

// SetLinePrefix sets a prefix for each rendered line.
func (r *Renderer) SetLinePrefix(prefix string) {
	r.linePrefix = prefix
}

// Write consumes streaming chunks and renders completed lines.
func (r *Renderer) Write(chunk string) {
	r.buffer.WriteString(chunk)
	for {
		data := r.buffer.String()
		idx := strings.IndexByte(data, '\n')
		if idx == -1 {
			return
		}
		line := data[:idx]
		r.renderLine(line)
		r.buffer.Reset()
		r.buffer.WriteString(data[idx+1:])
	}
}

// Flush renders any remaining buffered content.
func (r *Renderer) Flush() {
	if r.buffer.Len() == 0 {
		return
	}
	r.renderLine(r.buffer.String())
	r.buffer.Reset()
}

func (r *Renderer) renderLine(line string) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "```") {
		r.flushTable()
		if r.inCode {
			r.inCode = false
			r.codeLang = ""
			r.printLine("```", color.New(color.FgMagenta, color.Bold))
			return
		}

		r.inCode = true
		r.codeLang = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
		if r.codeLang != "" {
			r.printLine("```"+r.codeLang, color.New(color.FgMagenta, color.Bold))
		} else {
			r.printLine("```", color.New(color.FgMagenta, color.Bold))
		}
		return
	}

	if r.inCode {
		r.flushTable()
		r.printLine(line, color.New(color.FgHiBlue))
		return
	}

	if trimmed == "" {
		r.flushTable()
		r.printLine(line, nil)
		return
	}

	if isTableLine(trimmed) {
		r.tableLines = append(r.tableLines, trimmed)
		return
	}

	r.flushTable()

	switch {
	case isHorizontalRule(trimmed):
		r.printLine(strings.Repeat("─", 48), nil)
	case strings.HasPrefix(trimmed, "### "):
		r.printLine(renderInline(trimmed), color.New(color.FgCyan, color.Bold))
	case strings.HasPrefix(trimmed, "## "):
		r.printLine(renderInline(trimmed), color.New(color.FgGreen, color.Bold))
	case strings.HasPrefix(trimmed, "# "):
		r.printLine(renderInline(trimmed), color.New(color.FgYellow, color.Bold))
	case isBlockQuote(trimmed):
		r.renderBlockQuote(line)
	case isListItem(line):
		r.renderList(line)
	case isNumberedList(line):
		r.renderNumberedList(line)
	default:
		r.printLine(renderInline(line), nil)
	}
}

func isTableLine(line string) bool {
	if !strings.Contains(line, "|") {
		return false
	}
	return true
}

func isHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	trimmed = strings.ReplaceAll(trimmed, " ", "")
	for _, r := range trimmed {
		if r != '-' && r != '_' && r != '*' {
			return false
		}
	}
	return true
}

func isListItem(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")
}

func isNumberedList(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	dot := strings.Index(trimmed, ".")
	if dot <= 0 {
		return false
	}
	for i := 0; i < dot; i++ {
		if trimmed[i] < '0' || trimmed[i] > '9' {
			return false
		}
	}
	return len(trimmed) > dot+1 && trimmed[dot+1] == ' '
}

func isBlockQuote(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " \t"), ">")
}

func isSeparatorLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	cells := strings.Split(trimmed, "|")
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			continue
		}
		for _, r := range cell {
			if r != '-' && r != ':' {
				return false
			}
		}
	}
	return true
}

func parseTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func (r *Renderer) flushTable() {
	if len(r.tableLines) == 0 {
		return
	}

	var rows [][]string
	for _, line := range r.tableLines {
		if isSeparatorLine(line) {
			continue
		}
		rows = append(rows, parseTableRow(line))
	}
	r.tableLines = nil
	if len(rows) == 0 {
		return
	}

	colCount := 0
	for _, row := range rows {
		if len(row) > colCount {
			colCount = len(row)
		}
	}
	if colCount == 0 {
		return
	}

	widths := make([]int, colCount)
	for _, row := range rows {
		for i := 0; i < colCount; i++ {
			val := ""
			if i < len(row) {
				val = row[i]
			}
			if len(val) > widths[i] {
				widths[i] = len(val)
			}
		}
	}

	border := func() string {
		var b strings.Builder
		b.WriteString("+")
		for _, w := range widths {
			b.WriteString(strings.Repeat("-", w+2))
			b.WriteString("+")
		}
		return b.String()
	}

	r.printLine(border(), nil)
	for idx, row := range rows {
		r.renderTableRow(row, widths, idx == 0)
		if idx == 0 && len(rows) > 1 {
			r.printLine(border(), nil)
		}
	}
	r.printLine(border(), nil)
}

func (r *Renderer) renderTableRow(row []string, widths []int, header bool) {
	var b strings.Builder
	b.WriteString("|")
	for i := 0; i < len(widths); i++ {
		val := ""
		if i < len(row) {
			val = row[i]
		}
		padding := widths[i] - len(val)
		b.WriteString(" ")
		b.WriteString(val)
		b.WriteString(strings.Repeat(" ", padding))
		b.WriteString(" |")
	}
	if header {
		r.printLine(b.String(), color.New(color.FgHiWhite, color.Bold))
	} else {
		r.printLine(b.String(), nil)
	}
}

func (r *Renderer) renderList(line string) {
	trimmed := strings.TrimLeft(line, " \t")
	indent := len(line) - len(trimmed)
	level := indent / 2
	if level < 0 {
		level = 0
	}
	bullet := "•"
	if level%2 == 1 {
		bullet = "◦"
	}
	content := strings.TrimSpace(trimmed[1:])
	prefix := strings.Repeat("  ", level)
	if checkbox, ok := renderCheckbox(content); ok {
		r.printLine(prefix+bullet+" "+checkbox, color.New(color.FgHiWhite))
		return
	}
	r.printLine(prefix+bullet+" "+renderInline(content), color.New(color.FgHiWhite))
}

func (r *Renderer) renderNumberedList(line string) {
	trimmed := strings.TrimLeft(line, " \t")
	indent := len(line) - len(trimmed)
	level := indent / 2
	dot := strings.Index(trimmed, ".")
	if dot == -1 {
		r.printLine(renderInline(line), nil)
		return
	}
	number := trimmed[:dot]
	content := strings.TrimSpace(trimmed[dot+1:])
	prefix := strings.Repeat("  ", level)
	if checkbox, ok := renderCheckbox(content); ok {
		r.printLine(prefix+number+". "+checkbox, color.New(color.FgHiWhite))
		return
	}
	r.printLine(prefix+number+". "+renderInline(content), color.New(color.FgHiWhite))
}

func (r *Renderer) renderBlockQuote(line string) {
	trimmed := strings.TrimLeft(line, " \t")
	content := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
	prefix := color.New(color.FgHiBlack).Sprint("| ")
	r.printLine(prefix+renderInline(content), nil)
}

func (r *Renderer) printLine(text string, style *color.Color) {
	if r.linePrefix != "" {
		fmt.Fprint(r.writer, r.linePrefix)
	}
	if style != nil {
		style.Fprintln(r.writer, text)
		return
	}
	fmt.Fprintln(r.writer, text)
}

func renderInline(line string) string {
	var b strings.Builder
	var segment strings.Builder
	inCode := false
	inBold := false

	flush := func() {
		if segment.Len() == 0 {
			return
		}
		text := segment.String()
		switch {
		case inCode:
			b.WriteString(color.New(color.FgYellow).Sprint(text))
		case inBold:
			b.WriteString(color.New(color.Bold).Sprint(text))
		default:
			b.WriteString(text)
		}
		segment.Reset()
	}

	i := 0
	for i < len(line) {
		if strings.HasPrefix(line[i:], "**") && !inCode {
			flush()
			inBold = !inBold
			i += 2
			continue
		}
		if line[i] == '`' {
			flush()
			inCode = !inCode
			i++
			continue
		}
		if line[i] == '!' && i+1 < len(line) && line[i+1] == '[' {
			endText := strings.IndexByte(line[i+1:], ']')
			if endText > 0 {
				textEnd := i + 1 + endText
				if textEnd+1 < len(line) && line[textEnd+1] == '(' {
					endURL := strings.IndexByte(line[textEnd+2:], ')')
					if endURL >= 0 {
						flush()
						alt := line[i+2 : textEnd]
						url := line[textEnd+2 : textEnd+2+endURL]
						b.WriteString(color.New(color.FgHiBlack).Sprint("img: "))
						b.WriteString(color.New(color.FgCyan).Sprint(alt))
						b.WriteString(" ")
						b.WriteString(color.New(color.FgHiBlack).Sprint("(" + url + ")"))
						i = textEnd + 3 + endURL
						continue
					}
				}
			}
		}
		if line[i] == '[' {
			endText := strings.IndexByte(line[i:], ']')
			if endText > 0 {
				textEnd := i + endText
				if textEnd+1 < len(line) && line[textEnd+1] == '(' {
					endURL := strings.IndexByte(line[textEnd+2:], ')')
					if endURL >= 0 {
						flush()
						text := line[i+1 : textEnd]
						url := line[textEnd+2 : textEnd+2+endURL]
						b.WriteString(color.New(color.FgCyan, color.Underline).Sprint(text))
						b.WriteString(" ")
						b.WriteString(color.New(color.FgHiBlack).Sprint("(" + url + ")"))
						i = textEnd + 3 + endURL
						continue
					}
				}
			}
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		if r == utf8.RuneError && size == 1 {
			size = 1
		}
		segment.WriteString(line[i : i+size])
		i += size
	}
	flush()
	return b.String()
}

func renderCheckbox(content string) (string, bool) {
	lower := strings.ToLower(content)
	switch {
	case strings.HasPrefix(lower, "[x] "):
		rest := strings.TrimSpace(content[3:])
		check := color.New(color.FgGreen, color.Bold).Sprint("[x]")
		return check + " " + renderInline(rest), true
	case strings.HasPrefix(lower, "[ ] "):
		rest := strings.TrimSpace(content[3:])
		check := color.New(color.FgYellow).Sprint("[ ]")
		return check + " " + renderInline(rest), true
	default:
		return "", false
	}
}
