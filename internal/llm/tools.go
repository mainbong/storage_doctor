package llm

// Tool represents a function that can be called by the LLM
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

// ToolCall represents a function call request from the LLM
type ToolCall struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// GetTools returns the available tools for the LLM
func GetTools() []Tool {
	return []Tool{
		{
			Name:        "execute_command",
			Description: "쉘 명령어를 실행합니다. 명령어 실행 전에 사용자 승인이 필요합니다.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "실행할 쉘 명령어",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "명령어 실행 목적 설명",
					},
				},
				"required": []string{"command", "description"},
			},
		},
		{
			Name:        "read_file",
			Description: "파일을 읽고 내용을 반환합니다.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "읽을 파일 경로",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "write_file",
			Description: "파일을 작성하거나 수정합니다. 변경 전에 자동으로 백업이 생성됩니다.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "작성할 파일 경로",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "파일 내용",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "파일 수정 목적 설명",
					},
				},
				"required": []string{"path", "content", "description"},
			},
		},
		{
			Name:        "search_web",
			Description: "웹에서 정보를 검색합니다. 스토리지 문제 해결을 위한 유사 사례나 해결 방안을 찾을 때 사용합니다.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "검색 쿼리",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "monitor_log",
			Description: "로그 파일을 모니터링하거나 검색합니다.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "로그 파일 경로",
					},
					"action": map[string]interface{}{
						"type":        "string",
						"description": "동작: 'tail' (실시간 모니터링), 'search' (패턴 검색), 'filter' (레벨 필터링), 'summarize' (요약)",
					},
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "검색 패턴 (search 또는 filter 액션 시 필요)",
					},
				},
				"required": []string{"path", "action"},
			},
		},
		{
			Name:        "ask_user",
			Description: "사용자에게 추가 정보를 요청하거나 확인을 받습니다.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"question": map[string]interface{}{
						"type":        "string",
						"description": "사용자에게 물어볼 질문",
					},
				},
				"required": []string{"question"},
			},
		},
	}
}




