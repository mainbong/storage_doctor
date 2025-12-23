package main

import (
	"fmt"
	"strings"

	"github.com/mainbong/storage_doctor/internal/llm"
	"github.com/mainbong/storage_doctor/internal/shell"
)

func needsApproval(toolCall llm.ToolCall) bool {
	switch toolCall.Name {
	case "execute_command", "write_file":
		return !cfg.AutoApproveCommands && shellExec.GetApprovalMode() == shell.ApprovalModeManual
	default:
		return false
	}
}

func approvalOptionsCount(toolCall llm.ToolCall) int {
	if toolCall.Name == "execute_command" {
		return 3
	}
	return 2
}

func isAutoApproved(auto map[string]bool, toolCall llm.ToolCall) bool {
	if toolCall.Name != "execute_command" {
		return false
	}
	key := commandKeyFromTool(toolCall)
	return key != "" && auto[key]
}

func commandKeyFromApproval(req *approvalRequest) string {
	if req == nil {
		return ""
	}
	return commandKeyFromTool(req.tool)
}

func commandKeyFromTool(toolCall llm.ToolCall) string {
	command, _ := toolCall.Input["command"].(string)
	return commandKey(command)
}

func commandKey(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	var parts []string
	for _, f := range fields {
		if strings.HasPrefix(f, "-") {
			break
		}
		parts = append(parts, f)
	}
	if len(parts) == 0 {
		return fields[0]
	}
	return strings.Join(parts, " ")
}

func renderApprovalPromptWithSelection(req *approvalRequest, width int, approveIdx int) string {
	title, body := approvalContent(req.tool)
	options := approvalOptionsCount(req.tool)
	yesLabel := approvalOption.Render("[ 승인 ]")
	noLabel := approvalOption.Render("[ 취소 ]")
	autoLabel := approvalOption.Render("[ 자동 승인 ]")
	switch approveIdx {
	case 0:
		yesLabel = approvalActive.Render("[ 승인 ]")
	case 1:
		noLabel = approvalActive.Render("[ 취소 ]")
	case 2:
		autoLabel = approvalActive.Render("[ 자동 승인 ]")
	}
	choices := yesLabel + "  " + noLabel
	if options == 3 {
		choices = choices + "  " + autoLabel
	}
	hint := "y/n 또는 화살표 + Enter"
	if options == 3 {
		hint = "y/n/a 또는 화살표 + Enter"
	}
	content := approvalTitle.Render(title) + "\n" + body + "\n" + choices + "\n" + approvalHint.Render(hint)
	if width <= 0 {
		return approvalBox.Render(content)
	}
	return approvalBox.Width(min(width-2, 80)).Render(content)
}

func approvalContent(toolCall llm.ToolCall) (string, string) {
	switch toolCall.Name {
	case "execute_command":
		command, _ := toolCall.Input["command"].(string)
		desc, _ := toolCall.Input["description"].(string)
		key := commandKey(command)
		body := fmt.Sprintf("명령어: %s\n자동 승인 범위: %s", command, key)
		if desc != "" {
			body = fmt.Sprintf("목적: %s\n%s", desc, body)
		}
		return "명령어 실행 요청", body
	case "write_file":
		path, _ := toolCall.Input["path"].(string)
		desc, _ := toolCall.Input["description"].(string)
		body := fmt.Sprintf("파일: %s", path)
		if desc != "" {
			body = fmt.Sprintf("목적: %s\n%s", desc, body)
		}
		return "파일 수정 요청", body
	default:
		return "작업 실행 요청", toolCall.Name
	}
}
