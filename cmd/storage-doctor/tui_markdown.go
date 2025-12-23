package main

import "strings"

func renderMessages(messages []chatMessage, width int) string {
	if width <= 0 {
		width = 80
	}
	contentWidth := width - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	var b strings.Builder
	for _, msg := range messages {
		switch msg.role {
		case "user":
			b.WriteString(userLabelStyle.Render("사용자"))
			b.WriteString("\n")
			renderWrappedLines(&b, msg.content, contentWidth, func(line string) string {
				return userBubble.Render(line)
			})
			b.WriteString("\n")
		case "assistant":
			b.WriteString(assistantLabel.Render("답변"))
			b.WriteString("\n")
			renderMarkdownLines(&b, msg.content, contentWidth, func(line string) string {
				return assistantPrefix.Render("│ ") + assistantStyle.Render(line)
			})
			b.WriteString("\n")
		case "tool":
			b.WriteString(toolLabelStyle.Render("도구"))
			b.WriteString("\n")
			renderMarkdownLines(&b, msg.content, contentWidth, func(line string) string {
				return toolStyle.Render(line)
			})
			b.WriteString("\n")
		default:
			b.WriteString(systemLabel.Render("시스템"))
			b.WriteString("\n")
			renderMarkdownLines(&b, msg.content, contentWidth, func(line string) string {
				return systemStyle.Render(line)
			})
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderWrappedLines(b *strings.Builder, content string, width int, style func(string) string) {
	if content == "" {
		b.WriteString(style(" "))
		b.WriteString("\n")
		return
	}
	for _, line := range strings.Split(content, "\n") {
		wrapped := wrapText(line, width)
		if len(wrapped) == 0 {
			b.WriteString(style(" "))
			b.WriteString("\n")
			continue
		}
		for _, part := range wrapped {
			b.WriteString(style(part))
			b.WriteString("\n")
		}
	}
}

func renderMarkdownLines(b *strings.Builder, content string, width int, style func(string) string) {
	if content == "" {
		b.WriteString(style(" "))
		b.WriteString("\n")
		return
	}
	inCode := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
			b.WriteString(style(codeFenceStyle.Render(trimmed)))
			b.WriteString("\n")
			continue
		}
		if inCode {
			for _, part := range wrapText(line, width) {
				b.WriteString(style(codeBlockStyle.Render(part)))
				b.WriteString("\n")
			}
			continue
		}
		if trimmed == "" {
			b.WriteString(style(" "))
			b.WriteString("\n")
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "### "):
			renderMarkdownWrapped(b, strings.TrimPrefix(trimmed, "### "), width, func(s string) string {
				return style(heading3Style.Render(renderInlineStyle(s)))
			})
		case strings.HasPrefix(trimmed, "## "):
			renderMarkdownWrapped(b, strings.TrimPrefix(trimmed, "## "), width, func(s string) string {
				return style(heading2Style.Render(renderInlineStyle(s)))
			})
		case strings.HasPrefix(trimmed, "# "):
			renderMarkdownWrapped(b, strings.TrimPrefix(trimmed, "# "), width, func(s string) string {
				return style(heading1Style.Render(renderInlineStyle(s)))
			})
		case strings.HasPrefix(trimmed, "- "), strings.HasPrefix(trimmed, "* "):
			item := strings.TrimSpace(trimmed[2:])
			renderMarkdownWrapped(b, item, width-2, func(s string) string {
				return style(listBulletStyle.Render("• ") + renderInlineStyle(s))
			})
		case strings.HasPrefix(trimmed, ">"):
			quote := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
			renderMarkdownWrapped(b, quote, width-2, func(s string) string {
				return style(quoteStyle.Render("│ " + renderInlineStyle(s)))
			})
		default:
			renderMarkdownWrapped(b, line, width, func(s string) string {
				return style(renderInlineStyle(s))
			})
		}
	}
}

func renderMarkdownWrapped(b *strings.Builder, content string, width int, style func(string) string) {
	wrapped := wrapText(content, width)
	if len(wrapped) == 0 {
		b.WriteString(style(" "))
		b.WriteString("\n")
		return
	}
	for _, part := range wrapped {
		b.WriteString(style(part))
		b.WriteString("\n")
	}
}

func renderInlineStyle(text string) string {
	var out strings.Builder
	var segment strings.Builder
	inCode := false
	inBold := false
	flush := func() {
		if segment.Len() == 0 {
			return
		}
		part := segment.String()
		switch {
		case inCode:
			out.WriteString(inlineCodeStyle.Render(part))
		case inBold:
			out.WriteString(boldStyle.Render(part))
		default:
			out.WriteString(part)
		}
		segment.Reset()
	}

	i := 0
	for i < len(text) {
		if strings.HasPrefix(text[i:], "**") && !inCode {
			flush()
			inBold = !inBold
			i += 2
			continue
		}
		if text[i] == '`' {
			flush()
			inCode = !inCode
			i++
			continue
		}
		segment.WriteByte(text[i])
		i++
	}
	flush()
	return out.String()
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	var lines []string
	var line []rune
	count := 0
	for _, r := range []rune(text) {
		if r == '\n' {
			lines = append(lines, string(line))
			line = line[:0]
			count = 0
			continue
		}
		line = append(line, r)
		count++
		if count >= width {
			lines = append(lines, string(line))
			line = line[:0]
			count = 0
		}
	}
	if len(line) > 0 || len(lines) == 0 {
		lines = append(lines, string(line))
	}
	return lines
}
