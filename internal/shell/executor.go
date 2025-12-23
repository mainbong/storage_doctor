package shell

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ApprovalMode represents the command approval mode
type ApprovalMode int

const (
	ApprovalModeManual  ApprovalMode = iota // Ask for each command
	ApprovalModeAuto                        // Auto-approve all commands
	ApprovalModeSession                     // Auto-approve for current session
)

// Executor handles shell command execution with approval
type Executor struct {
	approvalMode    ApprovalMode
	workingDir      string
	commandExecutor CommandExecutor
}

// NewExecutor creates a new shell executor
func NewExecutor(workingDir string) *Executor {
	return NewExecutorWithCommandExecutor(workingDir, NewOSCommandExecutor())
}

// NewExecutorWithCommandExecutor creates a new shell executor with a custom CommandExecutor (for testing)
func NewExecutorWithCommandExecutor(workingDir string, executor CommandExecutor) *Executor {
	return &Executor{
		approvalMode:    ApprovalModeManual,
		workingDir:      workingDir,
		commandExecutor: executor,
	}
}

// SetApprovalMode sets the approval mode
func (e *Executor) SetApprovalMode(mode ApprovalMode) {
	e.approvalMode = mode
}

// GetApprovalMode returns the current approval mode
func (e *Executor) GetApprovalMode() ApprovalMode {
	return e.approvalMode
}

// Execute executes a shell command with approval
func (e *Executor) Execute(command string) (string, error) {
	// Check approval
	if e.approvalMode == ApprovalModeManual {
		approved, err := e.requestApproval(command)
		if err != nil {
			return "", err
		}
		if !approved {
			return "", fmt.Errorf("command execution cancelled by user")
		}
	}

	// Execute command
	return e.runCommand(command)
}

// runCommand runs a shell command and returns the output
func (e *Executor) runCommand(command string) (string, error) {
	output, err := e.commandExecutor.Execute(command, e.workingDir)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

// requestApproval requests user approval for a command
func (e *Executor) requestApproval(command string) (bool, error) {
	fmt.Printf("\n[명령어 실행 요청]\n")
	fmt.Printf("명령어: %s\n", command)
	fmt.Printf("실행하시겠습니까? [y/n/a/s]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))

	switch response {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	case "a", "always":
		e.approvalMode = ApprovalModeAuto
		fmt.Println("이제부터 모든 명령어를 자동으로 승인합니다.")
		return true, nil
	case "s", "session":
		e.approvalMode = ApprovalModeSession
		fmt.Println("이 세션 동안 모든 명령어를 자동으로 승인합니다.")
		return true, nil
	default:
		fmt.Println("잘못된 입력입니다. 'y' (예), 'n' (아니오), 'a' (항상 승인), 's' (세션 동안 승인) 중 하나를 입력하세요.")
		return e.requestApproval(command)
	}
}

// ExecuteSilent executes a command without approval (for internal use)
func (e *Executor) ExecuteSilent(command string) (string, error) {
	return e.runCommand(command)
}
