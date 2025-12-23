package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/mainbong/storage_doctor/internal/agent"
	"github.com/mainbong/storage_doctor/internal/chat"
	"github.com/mainbong/storage_doctor/internal/config"
	"github.com/mainbong/storage_doctor/internal/files"
	"github.com/mainbong/storage_doctor/internal/history"
	"github.com/mainbong/storage_doctor/internal/llm"
	"github.com/mainbong/storage_doctor/internal/logger"
	"github.com/mainbong/storage_doctor/internal/logs"
	"github.com/mainbong/storage_doctor/internal/search"
	"github.com/mainbong/storage_doctor/internal/shell"
	"github.com/mainbong/storage_doctor/internal/terminal"
)

var (
	cfg           *config.Config
	chatManager   *chat.Manager
	shellExec     *shell.Executor
	fileManager   *files.Manager
	searchMgr     *search.Manager
	historyMgr    *history.Manager
	agentInstance *agent.Agent
	skillMgr      *agent.SkillManager
	devMode       bool
	tuiEnabled    bool
)

var errExitRequested = errors.New("exit requested")

var rootCmd = &cobra.Command{
	Use:   "storage-doctor",
	Short: "스토리지 문제 해결을 위한 AI Assistant",
	Long:  "스토리지 관련 문제를 진단하고 해결하는 CLI 기반 AI Assistant입니다.",
	Run:   runREPL,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "버전 정보 출력",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("storage-doctor v0.1.0")
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "설정 관리",
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "설정 값 설정",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			fmt.Println("사용법: storage-doctor config set [key] [value]")
			return
		}
		if err := cfg.Set(args[0], args[1]); err != nil {
			fmt.Printf("설정 변경 실패: %v\n", err)
			return
		}
		if err := cfg.Save(); err != nil {
			fmt.Printf("설정 저장 실패: %v\n", err)
			return
		}
		fmt.Printf("설정 %s = %s\n", args[0], args[1])
	},
}

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "스킬 관리",
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "스킬 목록 조회",
	Run: func(cmd *cobra.Command, args []string) {
		skills := skillMgr.GetSkills()
		if len(skills) == 0 {
			fmt.Println("사용 가능한 스킬이 없습니다.")
			return
		}
		fmt.Println("사용 가능한 스킬:")
		for _, skill := range skills {
			fmt.Printf("  - %s: %s (%s)\n", skill.Name, skill.Description, skill.Path)
		}
	},
}

var skillsActivateCmd = &cobra.Command{
	Use:   "activate [name]",
	Short: "스킬 활성화",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("사용법: storage-doctor skills activate [name]")
			return
		}
		if err := agentInstance.ActivateSkill(args[0]); err != nil {
			fmt.Printf("스킬 활성화 실패: %v\n", err)
			return
		}
		fmt.Printf("스킬 '%s' 활성화 완료\n", args[0])
	},
}

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "세션 관리",
}

var sessionSaveCmd = &cobra.Command{
	Use:   "save [name]",
	Short: "현재 세션 저장",
	Run: func(cmd *cobra.Command, args []string) {
		name := "default"
		if len(args) > 0 {
			name = args[0]
		}
		if err := historyMgr.SaveSession(name); err != nil {
			fmt.Printf("세션 저장 실패: %v\n", err)
			return
		}
		fmt.Printf("세션 '%s' 저장 완료\n", name)
	},
}

var sessionLoadCmd = &cobra.Command{
	Use:   "load [session-id]",
	Short: "세션 로드",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("사용법: storage-doctor session load [session-id]")
			return
		}
		if err := historyMgr.LoadSession(args[0]); err != nil {
			fmt.Printf("세션 로드 실패: %v\n", err)
			return
		}
		fmt.Printf("세션 '%s' 로드 완료\n", args[0])
		actions := historyMgr.GetActions()
		if len(actions) == 0 {
			fmt.Println("복구된 작업 히스토리가 없습니다.")
			return
		}
		last := actions[len(actions)-1]
		switch last.Type {
		case history.ActionTypeCommand:
			fmt.Printf("복구된 작업 %d개 (최근: 명령어 %s)\n", len(actions), last.Command)
		case history.ActionTypeFile:
			fmt.Printf("복구된 작업 %d개 (최근: 파일 %s)\n", len(actions), last.FilePath)
		default:
			fmt.Printf("복구된 작업 %d개\n", len(actions))
		}
	},
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "세션 목록 조회",
	Run: func(cmd *cobra.Command, args []string) {
		sessions, err := historyMgr.ListSessions()
		if err != nil {
			fmt.Printf("세션 목록 조회 실패: %v\n", err)
			return
		}
		fmt.Println("저장된 세션:")
		for _, session := range sessions {
			fmt.Printf("  - %s (ID: %s, 생성: %s)\n", session.Name, session.ID, session.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	},
}

var sessionHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "현재 세션 작업 히스토리 조회",
	Run: func(cmd *cobra.Command, args []string) {
		actions := historyMgr.GetActions()
		if len(actions) == 0 {
			fmt.Println("현재 세션에 기록된 작업이 없습니다.")
			return
		}
		fmt.Println("현재 세션 작업 히스토리:")
		for i, action := range actions {
			switch action.Type {
			case history.ActionTypeCommand:
				fmt.Printf("  %d. [%s] 명령어: %s\n", i+1, action.Timestamp.Format("2006-01-02 15:04:05"), action.Command)
			case history.ActionTypeFile:
				fmt.Printf("  %d. [%s] 파일 수정: %s\n", i+1, action.Timestamp.Format("2006-01-02 15:04:05"), action.FilePath)
			default:
				fmt.Printf("  %d. [%s] 알 수 없는 작업 타입: %s\n", i+1, action.Timestamp.Format("2006-01-02 15:04:05"), action.Type)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(sessionCmd)
	rootCmd.AddCommand(skillsCmd)

	configCmd.AddCommand(configSetCmd)

	sessionCmd.AddCommand(sessionSaveCmd)
	sessionCmd.AddCommand(sessionLoadCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionHistoryCmd)

	skillsCmd.AddCommand(skillsListCmd)
	skillsCmd.AddCommand(skillsActivateCmd)

	// Add --dev flag
	rootCmd.Flags().BoolVar(&devMode, "dev", false, "개발 모드 활성화 (로그 파일을 현재 디렉토리에 저장)")
}

// ensureAPIKeys ensures that the required API key is set for the selected LLM provider
func ensureAPIKeys(cfg *config.Config) error {
	reader := bufio.NewReader(os.Stdin)

	switch cfg.LLMProvider {
	case "anthropic":
		if cfg.Anthropic.APIKey == "" {
			fmt.Printf("\n[설정 필요] Anthropic API 키가 설정되지 않았습니다.\n")
			fmt.Printf("1. 환경변수 ANTHROPIC_API_KEY 설정\n")
			fmt.Printf("2. 또는 아래에 직접 입력\n\n")
			fmt.Printf("Anthropic API Key: ")

			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "" {
				return fmt.Errorf("Anthropic API 키가 필요합니다")
			}

			cfg.Anthropic.APIKey = input
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("설정 저장 실패: %w", err)
			}
			fmt.Printf("✓ Anthropic API 키가 설정되었습니다. (저장 위치: %s)\n", config.GetConfigFile())
		}
	case "openai":
		if cfg.OpenAI.APIKey == "" {
			fmt.Printf("\n[설정 필요] OpenAI API 키가 설정되지 않았습니다.\n")
			fmt.Printf("1. 환경변수 OPENAI_API_KEY 설정\n")
			fmt.Printf("2. 또는 아래에 직접 입력\n\n")
			fmt.Printf("OpenAI API Key: ")

			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "" {
				return fmt.Errorf("OpenAI API 키가 필요합니다")
			}

			cfg.OpenAI.APIKey = input
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("설정 저장 실패: %w", err)
			}
			fmt.Printf("✓ OpenAI API 키가 설정되었습니다. (저장 위치: %s)\n", config.GetConfigFile())
		}
	default:
		return fmt.Errorf("알 수 없는 LLM 프로바이더: %s", cfg.LLMProvider)
	}
	return nil
}

func main() {
	var err error

	// Load configuration
	cfg, err = config.Load()
	if err != nil {
		fmt.Printf("설정 로드 실패: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logLevel := logger.INFO
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		logLevel = logger.DEBUG
	case "info":
		logLevel = logger.INFO
	case "warn":
		logLevel = logger.WARN
	case "error":
		logLevel = logger.ERROR
	}

	// Determine log directory based on dev mode
	logDir := cfg.LogDir
	if devMode {
		// In dev mode, use current working directory
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("현재 디렉토리 확인 실패: %v\n", err)
			os.Exit(1)
		}
		logDir = cwd
		fmt.Printf("[개발 모드] 로그 파일이 현재 디렉토리에 저장됩니다: %s\n", logDir)
	}

	if err := logger.Init(logDir, logLevel); err != nil {
		fmt.Printf("로거 초기화 실패: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	tuiEnabled = terminal.HasTTY()
	if !tuiEnabled {
		color.NoColor = true
	}

	runPreflightChecks()

	logger.Info("Storage Doctor 시작")
	logger.Debug("설정 로드 완료: LLM Provider=%s", cfg.LLMProvider)

	// Ensure API keys are set before initializing LLM provider
	if err := ensureAPIKeys(cfg); err != nil {
		logger.Error("API 키 설정 실패: %v", err)
		fmt.Printf("API 키 설정 실패: %v\n", err)
		os.Exit(1)
	}
	logger.Info("API 키 확인 완료")

	// Initialize LLM provider
	llmProvider, err := llm.NewProvider(cfg)
	if err != nil {
		logger.Error("LLM 프로바이더 초기화 실패: %v", err)
		fmt.Printf("LLM 프로바이더 초기화 실패: %v\n", err)
		os.Exit(1)
	}
	logger.Info("LLM 프로바이더 초기화 완료: %s", llmProvider.GetModel())

	// Initialize managers
	chatManager = chat.NewManager(llmProvider)
	logger.Debug("Chat Manager 초기화 완료")

	shellExec = shell.NewExecutor("")
	logger.Debug("Shell Executor 초기화 완료")

	fileManager = files.NewManager(cfg.BackupDir)
	logger.Debug("File Manager 초기화 완료: BackupDir=%s", cfg.BackupDir)

	searchMgr, err = search.NewManager(cfg)
	if err != nil {
		logger.Warn("검색 매니저 초기화 실패: %v (검색 기능 없이 계속)", err)
		fmt.Printf("검색 매니저 초기화 실패: %v\n", err)
		// Continue without search
		searchMgr = nil
	} else {
		logger.Debug("Search Manager 초기화 완료: Provider=%s", cfg.Search.Provider)
	}

	historyMgr, err = history.NewManager(cfg.SessionDir)
	if err != nil {
		logger.Error("히스토리 매니저 초기화 실패: %v", err)
		fmt.Printf("히스토리 매니저 초기화 실패: %v\n", err)
		os.Exit(1)
	}
	logger.Debug("History Manager 초기화 완료: SessionDir=%s", cfg.SessionDir)

	// Initialize Skill Manager
	skillsDir := filepath.Join(config.GetConfigDir(), "skills")
	skillMgr, err = agent.NewSkillManager(skillsDir)
	if err != nil {
		logger.Error("스킬 매니저 초기화 실패: %v", err)
		fmt.Printf("스킬 매니저 초기화 실패: %v\n", err)
		os.Exit(1)
	}
	logger.Info("스킬 매니저 초기화 완료: SkillsDir=%s", skillsDir)

	// Initialize Agent
	agentInstance = agent.NewAgent(llmProvider, chatManager, skillMgr)
	logger.Info("Agent 초기화 완료")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runREPL(cmd *cobra.Command, args []string) {
	color.Cyan("=== Storage Doctor AI Assistant ===\n")
	color.Yellow("스토리지 문제를 설명해주세요. 'exit' 또는 'quit'로 종료합니다.\n\n")

	ctx := context.Background()

	if tuiEnabled {
		if err := runTUI(); err != nil {
			color.Red("TUI 실행 실패: %v\n", err)
		}
		return
	}

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	var cancelMu sync.Mutex
	var currentCancel context.CancelFunc
	go func() {
		for range sigChan {
			cancelMu.Lock()
			cancel := currentCancel
			cancelMu.Unlock()
			if cancel != nil {
				cancel()
				continue
			}
			color.Yellow("\n종료합니다.\n")
			os.Exit(0)
		}
	}()

	reader := bufio.NewReader(os.Stdin)

	for {
		input, err := promptUserInput(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				color.Yellow("\n종료합니다.\n")
				break
			}
			fmt.Printf("입력 읽기 오류: %v\n", err)
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Check for exit commands
		if input == "exit" || input == "quit" || input == "q" {
			color.Yellow("\n종료합니다.\n")
			break
		}

		if handleOutputCommand(input) {
			continue
		}

		taskCtx, taskCancel := context.WithCancel(ctx)
		cancelMu.Lock()
		currentCancel = taskCancel
		cancelMu.Unlock()

		if tuiEnabled {
			printUserMessage(input)
		}

		// Process user input
		if err := processInput(taskCtx, input, reader); err != nil {
			if errors.Is(err, context.Canceled) {
				color.Yellow("\n[작업이 중단되었습니다]\n")
			} else if errors.Is(err, errExitRequested) {
				color.Yellow("\n종료합니다.\n")
				break
			} else {
				color.Red("오류: %v\n", err)
			}
		}
		taskCancel()
		cancelMu.Lock()
		currentCancel = nil
		cancelMu.Unlock()
	}
}

func promptUserInput(reader *bufio.Reader) (string, error) {
	if tuiEnabled {
		printInputDivider()
		color.New(color.FgHiBlack).Fprintln(os.Stdout, "입력")
		color.Green("> ")
		fmt.Fprint(os.Stdout, "\x1b[s")
		fmt.Fprint(os.Stdout, "\n")
		printInputDivider()
		printShortcutHint()
		fmt.Fprint(os.Stdout, "\x1b[u")
	} else {
		color.Green("> ")
	}
	input, err := readMultilineInput(reader)
	if tuiEnabled {
		fmt.Fprint(os.Stdout, "\n\n")
	}
	return input, err
}

func terminalWidth() int {
	const minWidth = 40
	const fallback = 80
	if cols := strings.TrimSpace(os.Getenv("COLUMNS")); cols != "" {
		if parsed, err := strconv.Atoi(cols); err == nil && parsed >= minWidth {
			return parsed
		}
	}
	return fallback
}

func printInputDivider() {
	width := terminalWidth()
	fmt.Println(strings.Repeat("-", width))
}

func printShortcutHint() {
	color.New(color.FgHiBlack).Fprintln(os.Stdout, "? 단축키 안내 (추가 예정)")
}

func printUserMessage(message string) {
	fmt.Println()
	color.New(color.FgHiBlack).Fprintln(os.Stdout, "사용자")
	bg := color.New(color.BgHiBlack, color.FgHiWhite)
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		if line == "" {
			bg.Fprintln(os.Stdout, " ")
			continue
		}
		bg.Fprintln(os.Stdout, " "+line+" ")
	}
	fmt.Println()
}

func printAssistantHeader() {
	fmt.Println()
	color.New(color.FgHiBlack).Fprintln(os.Stdout, "답변")
	fmt.Println(strings.Repeat("-", terminalWidth()))
}

func printAssistantFooter() {
	fmt.Println(strings.Repeat("-", terminalWidth()))
	fmt.Println()
}

func readMultilineInput(reader *bufio.Reader) (string, error) {
	var builder strings.Builder
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	builder.WriteString(line)
	for reader.Buffered() > 0 {
		next, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return "", readErr
		}
		builder.WriteString(next)
		if errors.Is(readErr, io.EOF) {
			break
		}
	}
	if errors.Is(err, io.EOF) && builder.Len() == 0 {
		return "", io.EOF
	}
	return strings.TrimRight(builder.String(), "\r\n"), nil
}

func processInput(ctx context.Context, userInput string, reader *bufio.Reader) error {
	logger.Info("사용자 입력 수신: %s", userInput)

	// Use Agent system for autonomous task execution
	if tuiEnabled {
		printAssistantHeader()
	} else {
		color.Cyan("\n[Agent]\n")
	}

	var renderer *terminal.Renderer
	if tuiEnabled {
		renderer = terminal.NewRenderer(os.Stdout)
		renderer.SetLinePrefix(color.New(color.FgHiBlack).Sprint("| "))
	}

	err := agentInstance.StreamTask(ctx, userInput, func(chunk string) {
		if renderer != nil {
			renderer.Write(chunk)
			return
		}
		fmt.Print(chunk)
	}, func(toolCall llm.ToolCall) (string, error) {
		logger.Debug("도구 호출: %s", toolCall.Name)
		result, err := executeToolCallForAgent(ctx, toolCall)
		if err != nil {
			logger.Error("도구 실행 실패: %s - %v", toolCall.Name, err)
		} else {
			logger.Debug("도구 실행 성공: %s", toolCall.Name)
		}
		return result, err
	})

	if renderer != nil {
		renderer.Flush()
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		logger.Error("Agent 작업 처리 실패: %v", err)
		color.Red("\n[오류] %v\n", err)
		return fmt.Errorf("Agent 작업 처리 실패: %w", err)
	}

	// Check if we got any response
	logger.Info("Agent 작업 완료")

	if tuiEnabled {
		printAssistantFooter()
	} else {
		fmt.Println()
	}

	// Ask user if they want to continue
	color.New(color.FgHiBlack).Fprintln(os.Stdout, "추가로 질문이 있으시면 입력해주세요. (엔터만 누르면 계속)")
	additionalInput, err := promptUserInput(reader)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	additionalInput = strings.TrimSpace(additionalInput)

	if additionalInput != "" {
		if additionalInput == "exit" || additionalInput == "quit" || additionalInput == "q" {
			return errExitRequested
		}
		return processInput(ctx, additionalInput, reader)
	}

	return nil
}

func handleOutputCommand(input string) bool {
	fields := strings.Fields(strings.ToLower(input))
	if len(fields) < 2 || fields[0] != "show" || fields[1] != "output" {
		return false
	}

	showAll := len(fields) >= 3 && fields[2] == "all"
	actions := historyMgr.GetActions()
	if len(actions) == 0 {
		fmt.Println("명령어 출력이 없습니다.")
		return true
	}

	if showAll {
		fmt.Println("명령어 출력 전체:")
		for _, action := range actions {
			if action.Type != history.ActionTypeCommand || action.Output == "" {
				continue
			}
			fmt.Printf("\n[명령어] %s\n", action.Command)
			fmt.Printf("%s\n", action.Output)
		}
		return true
	}

	for i := len(actions) - 1; i >= 0; i-- {
		action := actions[i]
		if action.Type == history.ActionTypeCommand && action.Output != "" {
			fmt.Println("최근 명령어 출력:")
			fmt.Printf("%s\n", action.Output)
			return true
		}
	}

	fmt.Println("명령어 출력이 없습니다.")
	return true
}

// executeToolCallForAgent executes a tool call and returns result for agent
func executeToolCallForAgent(ctx context.Context, toolCall llm.ToolCall) (string, error) {
	result, success, err := handleToolCall(ctx, toolCall, false, false)
	if !success {
		return result, err
	}
	return result, nil
}

func executeToolCallForAgentApproved(ctx context.Context, toolCall llm.ToolCall, approved bool) (string, error) {
	result, success, err := handleToolCall(ctx, toolCall, true, approved)
	if !success {
		return result, err
	}
	return result, nil
}

func executeToolCall(ctx context.Context, toolCall llm.ToolCall) error {
	result, success, err := handleToolCall(ctx, toolCall, false, false)

	// Add tool result to chat context
	toolResult := chat.FormatToolCall(toolCall.Name, result, success)
	chatManager.AddMessage("user", toolResult)

	color.Green("\n[도구 실행 완료]\n")
	if !success {
		color.Red("%s\n", result)
	} else {
		color.Cyan("%s\n", result)
	}

	return err
}

func handleToolCall(ctx context.Context, toolCall llm.ToolCall, quiet bool, approved bool) (string, bool, error) {
	var result string
	var success bool
	var err error

	switch toolCall.Name {
	case "execute_command":
		command, ok := toolCall.Input["command"].(string)
		if !ok {
			return "", false, fmt.Errorf("invalid command parameter")
		}
		description, _ := toolCall.Input["description"].(string)

		if !quiet {
			color.Yellow("\n[명령어 실행 요청]\n")
			if description != "" {
				color.Cyan("목적: %s\n", description)
			}
			color.Cyan("명령어: %s\n", command)
		}

		// Always ask for approval unless auto-approve is set
		if !cfg.AutoApproveCommands && shellExec.GetApprovalMode() == shell.ApprovalModeManual {
			if !approved {
				if quiet {
					return "", false, fmt.Errorf("승인 필요")
				}
				// Request approval
				reader := bufio.NewReader(os.Stdin)
				fmt.Printf("실행하시겠습니까? [y/n/a/s]: ")
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))

				switch response {
				case "y", "yes":
					// Continue
				case "n", "no":
					return "", false, fmt.Errorf("사용자가 명령어 실행을 취소했습니다")
				case "a", "always":
					cfg.AutoApproveCommands = true
					cfg.Save()
					color.Green("이제부터 모든 명령어를 자동으로 승인합니다.\n")
				case "s", "session":
					shellExec.SetApprovalMode(shell.ApprovalModeSession)
					color.Green("이 세션 동안 모든 명령어를 자동으로 승인합니다.\n")
				default:
					return "", false, fmt.Errorf("잘못된 입력입니다")
				}
			}
		}

		output, err := shellExec.ExecuteSilent(command)
		if err != nil {
			result = fmt.Sprintf("명령어 실행 실패: %v\n출력: %s", err, output)
			success = false
		} else {
			result = fmt.Sprintf("명령어 실행 성공\n출력:\n%s", output)
			success = true
			historyMgr.AddCommandAction(command, output)
			if !quiet {
				displayCommandOutput(output)
			}
			if err := historyMgr.SaveSession(""); err != nil {
				logger.Warn("세션 자동 저장 실패: %v", err)
			}
		}

	case "read_file":
		path, ok := toolCall.Input["path"].(string)
		if !ok {
			return "", false, fmt.Errorf("invalid path parameter")
		}

		content, err := fileManager.ReadFile(path)
		if err != nil {
			result = fmt.Sprintf("파일 읽기 실패: %v", err)
			success = false
		} else {
			result = fmt.Sprintf("파일 내용:\n%s", content)
			success = true
		}

	case "write_file":
		path, ok := toolCall.Input["path"].(string)
		if !ok {
			return "", false, fmt.Errorf("invalid path parameter")
		}
		content, ok := toolCall.Input["content"].(string)
		if !ok {
			return "", false, fmt.Errorf("invalid content parameter")
		}
		description, _ := toolCall.Input["description"].(string)

		if !quiet {
			color.Yellow("\n[파일 수정 요청]\n")
			if description != "" {
				color.Cyan("목적: %s\n", description)
			}
			color.Cyan("파일: %s\n", path)
		}

		// Read old content for backup
		oldContent, _ := fileManager.ReadFile(path)

		// Ask for approval
		if !approved {
			if quiet {
				return "", false, fmt.Errorf("승인 필요")
			}
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("파일을 수정하시겠습니까? [y/n]: ")
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "y" && response != "yes" {
				return "", false, fmt.Errorf("사용자가 파일 수정을 취소했습니다")
			}
		}

		err := fileManager.WriteFile(path, content)
		if err != nil {
			result = fmt.Sprintf("파일 쓰기 실패: %v", err)
			success = false
		} else {
			result = fmt.Sprintf("파일 수정 성공 (백업 생성됨)")
			success = true
			historyMgr.AddFileAction(path, oldContent, content)
			if err := historyMgr.SaveSession(""); err != nil {
				logger.Warn("세션 자동 저장 실패: %v", err)
			}
		}

	case "search_web":
		query, ok := toolCall.Input["query"].(string)
		if !ok {
			return "", false, fmt.Errorf("invalid query parameter")
		}

		if searchMgr == nil {
			return "", false, fmt.Errorf("검색 기능이 사용 불가능합니다")
		}

		if !quiet {
			color.Yellow("\n[웹 검색 중...]\n")
		}
		results, err := searchMgr.Search(ctx, query, 5)
		if err != nil {
			result = fmt.Sprintf("검색 실패: %v", err)
			success = false
		} else {
			result = searchMgr.FormatResults(results)
			success = true
		}

	case "monitor_log":
		path, ok := toolCall.Input["path"].(string)
		if !ok {
			return "", false, fmt.Errorf("invalid path parameter")
		}
		action, ok := toolCall.Input["action"].(string)
		if !ok {
			return "", false, fmt.Errorf("invalid action parameter")
		}

		monitor, err := logs.NewMonitor(path)
		if err != nil {
			return "", false, fmt.Errorf("로그 모니터 생성 실패: %w", err)
		}
		defer monitor.Close()

		switch action {
		case "tail":
			if quiet {
				return "", false, fmt.Errorf("TUI 모드에서는 로그 실시간 모니터링 출력을 지원하지 않습니다")
			}
			color.Yellow("\n[로그 실시간 모니터링 - Ctrl+C로 중지]\n")
			ctx, cancel := context.WithCancel(ctx)
			go func() {
				reader := bufio.NewReader(os.Stdin)
				reader.ReadString('\n')
				cancel()
			}()
			err = monitor.Tail(ctx, func(line string) {
				fmt.Println(line)
			})
			if err != nil && err != context.Canceled {
				result = fmt.Sprintf("로그 모니터링 실패: %v", err)
				success = false
			} else {
				result = "로그 모니터링 완료"
				success = true
			}
		case "search":
			pattern, _ := toolCall.Input["pattern"].(string)
			matches, err := monitor.Search(pattern)
			if err != nil {
				result = fmt.Sprintf("검색 실패: %v", err)
				success = false
			} else {
				result = fmt.Sprintf("검색 결과 (%d개):\n%s", len(matches), strings.Join(matches, "\n"))
				success = true
			}
		case "filter":
			pattern, _ := toolCall.Input["pattern"].(string)
			matches, err := monitor.Filter(pattern)
			if err != nil {
				result = fmt.Sprintf("필터링 실패: %v", err)
				success = false
			} else {
				result = fmt.Sprintf("필터링 결과 (%d개):\n%s", len(matches), strings.Join(matches, "\n"))
				success = true
			}
		case "summarize":
			stats, err := monitor.Summarize()
			if err != nil {
				result = fmt.Sprintf("요약 실패: %v", err)
				success = false
			} else {
				result = fmt.Sprintf("로그 요약:\n총 라인: %v\n에러: %v\n경고: %v\n정보: %v",
					stats["total_lines"], stats["error_count"], stats["warn_count"], stats["info_count"])
				success = true
			}
		default:
			return "", false, fmt.Errorf("unknown action: %s", action)
		}

	case "ask_user":
		question, ok := toolCall.Input["question"].(string)
		if !ok {
			return "", false, fmt.Errorf("invalid question parameter")
		}

		color.Yellow("\n[질문]\n")
		color.Cyan("%s\n", question)
		fmt.Printf("답변: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		result = fmt.Sprintf("사용자 답변: %s", answer)
		success = true

		chatManager.AddMessage("user", fmt.Sprintf("질문: %s\n답변: %s", question, answer))

	default:
		return "", false, fmt.Errorf("unknown tool: %s", toolCall.Name)
	}

	return result, success, err
}

func displayCommandOutput(output string) {
	if strings.TrimSpace(output) == "" {
		return
	}

	if !tuiEnabled {
		fmt.Printf("\n[명령어 출력]\n%s\n", output)
		return
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	maxLines := 20
	truncated := false
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		truncated = true
	}

	fmt.Println("\n[명령어 출력]")
	for _, line := range lines {
		fmt.Println(line)
	}
	if truncated {
		fmt.Printf("... (출력 일부 숨김, 'show output'으로 전체 확인)\n")
	}
}

func runPreflightChecks() {
	term := strings.ToLower(strings.TrimSpace(os.Getenv("TERM")))
	if term == "" || term == "dumb" {
		tuiEnabled = false
		color.NoColor = true
		logger.Warn("터미널 기능 제한: TERM=%q (TUI/컬러 비활성화)", term)
	}

	if os.Getenv("LC_ALL") == "" && os.Getenv("LANG") == "" {
		logger.Warn("로케일 설정이 비어 있습니다. UTF-8 로케일을 권장합니다 (예: LANG=C.UTF-8)")
	}

	if _, err := exec.LookPath("kubectl"); err != nil {
		logger.Warn("kubectl을 찾을 수 없습니다. 클러스터 진단 명령이 실패할 수 있습니다.")
	}

	checkKubeconfig()
	checkWritableDir(cfg.SessionDir, "session_dir")
	checkWritableDir(cfg.BackupDir, "backup_dir")
	checkWritableDir(cfg.LogDir, "log_dir")
}

func checkKubeconfig() {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		if _, err := os.Stat(kubeconfig); err != nil {
			logger.Warn("KUBECONFIG 경로를 찾을 수 없습니다: %s", kubeconfig)
		}
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		logger.Warn("홈 디렉토리 확인 실패: %v", err)
		return
	}
	defaultConfig := filepath.Join(home, ".kube", "config")
	if _, err := os.Stat(defaultConfig); err != nil {
		logger.Warn("기본 kubeconfig 파일이 없습니다: %s", defaultConfig)
	}
}

func checkWritableDir(path, label string) {
	if strings.TrimSpace(path) == "" {
		logger.Warn("%s 경로가 비어 있습니다.", label)
		return
	}
	testFile := filepath.Join(path, fmt.Sprintf(".writecheck-%d", time.Now().UnixNano()))
	if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
		logger.Warn("%s 경로에 쓰기 실패: %s (%v)", label, path, err)
		return
	}
	_ = os.Remove(testFile)
}
