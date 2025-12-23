package shell

// MockCommandExecutor is a mock implementation of CommandExecutor for testing
type MockCommandExecutor struct {
	responses map[string]string
	errors    map[string]error
	commands  []string
}

// NewMockCommandExecutor creates a new MockCommandExecutor instance
func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		responses: make(map[string]string),
		errors:    make(map[string]error),
		commands:  make([]string, 0),
	}
}

// SetResponse sets a mock response for a command
func (m *MockCommandExecutor) SetResponse(command string, output string) {
	m.responses[command] = output
}

// SetError sets an error to return for a command
func (m *MockCommandExecutor) SetError(command string, err error) {
	m.errors[command] = err
}

// GetCommands returns all commands executed
func (m *MockCommandExecutor) GetCommands() []string {
	return m.commands
}

// ClearCommands clears the command history
func (m *MockCommandExecutor) ClearCommands() {
	m.commands = make([]string, 0)
}

func (m *MockCommandExecutor) Execute(command string, dir string) ([]byte, error) {
	// Store command
	m.commands = append(m.commands, command)

	// Check for error
	if err, ok := m.errors[command]; ok {
		return nil, err
	}

	// Check for response
	if output, ok := m.responses[command]; ok {
		return []byte(output), nil
	}

	// Default: return empty output
	return []byte(""), nil
}

