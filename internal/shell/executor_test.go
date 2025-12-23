package shell

import (
	"errors"
	"strings"
	"testing"
)

func TestNewExecutor(t *testing.T) {
	executor := NewExecutor("/test/dir")
	if executor == nil {
		t.Fatal("NewExecutor() returned nil")
	}
	if executor.workingDir != "/test/dir" {
		t.Errorf("Expected workingDir '/test/dir', got '%s'", executor.workingDir)
	}
	if executor.approvalMode != ApprovalModeManual {
		t.Errorf("Expected ApprovalModeManual, got %v", executor.approvalMode)
	}
}

func TestSetApprovalMode(t *testing.T) {
	executor := NewExecutor("")
	
	executor.SetApprovalMode(ApprovalModeAuto)
	if executor.GetApprovalMode() != ApprovalModeAuto {
		t.Errorf("Expected ApprovalModeAuto, got %v", executor.GetApprovalMode())
	}

	executor.SetApprovalMode(ApprovalModeSession)
	if executor.GetApprovalMode() != ApprovalModeSession {
		t.Errorf("Expected ApprovalModeSession, got %v", executor.GetApprovalMode())
	}
}

func TestExecute_AutoMode(t *testing.T) {
	mockExecutor := NewMockCommandExecutor()
	executor := NewExecutorWithCommandExecutor("", mockExecutor)
	executor.SetApprovalMode(ApprovalModeAuto)

	testCommand := "echo test"
	expectedOutput := "test output"
	mockExecutor.SetResponse(testCommand, expectedOutput)

	output, err := executor.Execute(testCommand)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	if output != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, output)
	}

	// Verify command was executed
	commands := mockExecutor.GetCommands()
	if len(commands) != 1 || commands[0] != testCommand {
		t.Errorf("Expected command '%s' to be executed, got %v", testCommand, commands)
	}
}

func TestExecute_SessionMode(t *testing.T) {
	mockExecutor := NewMockCommandExecutor()
	executor := NewExecutorWithCommandExecutor("", mockExecutor)
	executor.SetApprovalMode(ApprovalModeSession)

	testCommand := "echo test"
	expectedOutput := "test output"
	mockExecutor.SetResponse(testCommand, expectedOutput)

	output, err := executor.Execute(testCommand)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	if output != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, output)
	}
}

func TestExecute_CommandError(t *testing.T) {
	mockExecutor := NewMockCommandExecutor()
	executor := NewExecutorWithCommandExecutor("", mockExecutor)
	executor.SetApprovalMode(ApprovalModeAuto)

	testCommand := "false"
	testError := errors.New("command failed")
	mockExecutor.SetError(testCommand, testError)

	output, err := executor.Execute(testCommand)
	if err == nil {
		t.Error("Expected error for command failure, got nil")
	}
	if !strings.Contains(err.Error(), "command failed") {
		t.Errorf("Expected error message to contain 'command failed', got '%v'", err)
	}
	// Output may be empty when command executor returns error
	_ = output
}

func TestExecuteSilent(t *testing.T) {
	mockExecutor := NewMockCommandExecutor()
	executor := NewExecutorWithCommandExecutor("", mockExecutor)

	testCommand := "echo test"
	expectedOutput := "test output"
	mockExecutor.SetResponse(testCommand, expectedOutput)

	output, err := executor.ExecuteSilent(testCommand)
	if err != nil {
		t.Fatalf("ExecuteSilent() failed: %v", err)
	}

	if output != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, output)
	}

	// Verify command was executed
	commands := mockExecutor.GetCommands()
	if len(commands) != 1 || commands[0] != testCommand {
		t.Errorf("Expected command '%s' to be executed, got %v", testCommand, commands)
	}
}

func TestExecute_WorkingDir(t *testing.T) {
	mockExecutor := NewMockCommandExecutor()
	workingDir := "/test/dir"
	executor := NewExecutorWithCommandExecutor(workingDir, mockExecutor)
	executor.SetApprovalMode(ApprovalModeAuto)

	testCommand := "pwd"
	mockExecutor.SetResponse(testCommand, "/test/dir")

	_, err := executor.Execute(testCommand)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Note: MockCommandExecutor doesn't verify working dir, but we can verify it's set
	if executor.workingDir != workingDir {
		t.Errorf("Expected workingDir '%s', got '%s'", workingDir, executor.workingDir)
	}
}

