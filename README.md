# Storage Doctor

스토리지 관련 문제를 해결해주기 위한 CLI 기반 AI Assistant

## 기능

- **AI 기반 문제 진단**: Anthropic Claude 또는 OpenAI GPT를 사용한 스토리지 문제 분석
- **웹 검색 통합**: 유사 사례 및 해결 방안 자동 검색
- **쉘 명령어 실행**: 문제 진단 및 해결을 위한 명령어 실행 (승인 시스템 포함)
- **파일 작업**: 설정 파일 읽기/쓰기/편집 (YAML, JSON, TOML 지원)
- **로그 모니터링**: 실시간 로그 tail 및 패턴 검색
- **작업 히스토리**: 모든 작업 기록 및 롤백 기능
- **세션 관리**: 대화 내용 및 작업 상태 저장/로드
- **TUI 대화 모드**: 터미널 스크롤 흐름에서 입력/응답/도구 출력이 시간 순서대로 표시

## 설치

```bash
git clone https://github.com/mainbong/storage_doctor.git
cd storage_doctor
go mod tidy
go build -o storage-doctor ./cmd/storage-doctor
```

## 설정

처음 실행 시 `~/.storage-doctor/config.json` 파일이 생성됩니다.

필수 설정:
- LLM Provider (anthropic 또는 openai)
- API Key

선택 설정:
- 검색 Provider (google, serper, duckduckgo)
- 검색 API Key

설정 예시:
```json
{
  "llm_provider": "anthropic",
  "anthropic": {
    "api_key": "your-api-key",
    "model": "claude-3-5-sonnet-20241022"
  },
  "openai": {
    "api_key": "",
    "model": "gpt-4-turbo-preview"
  },
  "search": {
    "provider": "duckduckgo",
    "google": {
      "api_key": "",
      "cx": ""
    },
    "serper": {
      "api_key": ""
    }
  },
  "auto_approve_commands": false,
  "session_dir": "~/.storage-doctor/sessions",
  "backup_dir": "~/.storage-doctor/backups"
}
```

## 사용법

### 기본 사용

```bash
storage-doctor
```

대화형 모드로 진입합니다. 스토리지 문제를 설명하면 Assistant가 진단하고 해결 방안을 제시합니다.

### TUI 단축키

- `Enter`: 전송
- `Shift+Enter`: 줄바꿈
- `Ctrl+C`: 종료

### 승인 UI (TUI)

명령어 실행 요청 시 선택 UI가 나타납니다:
- 화살표 + Enter 또는 `y/n/a`
- `승인`: 해당 명령 실행
- `취소`: 실행 취소 후 추가 요청 입력
- `자동 승인`: 현재 세션에서 해당 **키 명령어**(옵션 제외) 자동 승인

### 세션 관리

```bash
# 현재 세션 저장
storage-doctor session save my-session

# 세션 로드
storage-doctor session load <session-id>

# 세션 목록 조회
storage-doctor session list
```

### 명령어 승인

Assistant가 명령어를 제안하면 다음 옵션을 선택할 수 있습니다:
- `y` 또는 `yes`: 명령어 실행
- `n` 또는 `no`: 명령어 취소
- `a` 또는 `always`: 모든 명령어 자동 승인
- `s` 또는 `session`: 현재 세션 동안 자동 승인

## 아키텍처

- `cmd/storage-doctor/`: CLI 진입점
- `cmd/storage-doctor/tui_*.go`: TUI (ELM 스타일 구조 분리)
- `internal/llm/`: LLM Provider (Anthropic, OpenAI)
- `internal/chat/`: 대화 관리 및 컨텍스트 요약
- `internal/shell/`: 쉘 명령어 실행 및 승인 시스템
- `internal/search/`: 웹 검색 API 클라이언트
- `internal/files/`: 파일 읽기/쓰기/편집
- `internal/logs/`: 로그 파일 모니터링
- `internal/history/`: 작업 히스토리 및 세션 관리
- `internal/config/`: 설정 관리

## 라이선스

MIT
