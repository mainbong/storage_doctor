package shell

import "os/exec"

// CommandExecutor abstracts command execution for testability
type CommandExecutor interface {
	Execute(command string, dir string) ([]byte, error)
}

// OSCommandExecutor implements CommandExecutor using os/exec
type OSCommandExecutor struct{}

// NewOSCommandExecutor creates a new OSCommandExecutor instance
func NewOSCommandExecutor() *OSCommandExecutor {
	return &OSCommandExecutor{}
}

func (e *OSCommandExecutor) Execute(command string, dir string) ([]byte, error) {
	cmd := exec.Command("sh", "-c", command)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd.CombinedOutput()
}

