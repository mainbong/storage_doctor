package main

import (
	"fmt"
	"strings"

	"github.com/mainbong/storage_doctor/internal/llm"
)

func formatToolDisplay(toolCall llm.ToolCall, result string, err error) string {
	switch toolCall.Name {
	case "execute_command":
		command, _ := toolCall.Input["command"].(string)
		output := extractCommandOutput(result)
		output = truncateOutput(output, 12, 2000)
		status := "성공"
		if err != nil {
			status = fmt.Sprintf("오류: %v", err)
		}
		if strings.TrimSpace(output) == "" {
			output = "(출력 없음)"
		}
		return fmt.Sprintf("명령어: %s\n상태: %s\n출력:\n%s", command, status, output)
	default:
		if err != nil {
			return fmt.Sprintf("%s\n오류: %v\n%s", toolCall.Name, err, strings.TrimSpace(result))
		}
		return fmt.Sprintf("%s\n%s", toolCall.Name, strings.TrimSpace(result))
	}
}

func extractCommandOutput(result string) string {
	if result == "" {
		return ""
	}
	marker := "출력:\n"
	idx := strings.Index(result, marker)
	if idx == -1 {
		return result
	}
	return result[idx+len(marker):]
}

func truncateOutput(text string, maxLines int, maxChars int) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, "... (출력 일부 숨김)")
	}
	out := strings.Join(lines, "\n")
	if maxChars > 0 && len(out) > maxChars {
		out = out[:maxChars] + "\n... (출력 일부 숨김)"
	}
	return out
}
