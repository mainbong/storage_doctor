package history

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/mainbong/storage_doctor/internal/filesystem"
)

// ActionType represents the type of action
type ActionType string

const (
	ActionTypeCommand ActionType = "command"
	ActionTypeFile    ActionType = "file"
)

// Action represents a single action in history
type Action struct {
	ID        string     `json:"id"`
	Type      ActionType `json:"type"`
	Timestamp time.Time  `json:"timestamp"`
	Command   string     `json:"command,omitempty"`   // For command actions
	FilePath  string     `json:"file_path,omitempty"` // For file actions
	OldValue  string     `json:"old_value,omitempty"` // For file actions (backup)
	NewValue  string     `json:"new_value,omitempty"` // For file actions
	Output    string     `json:"output,omitempty"`    // For command actions
}

// Session represents a session with its history
type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Actions   []Action  `json:"actions"`
}

// Manager manages action history and sessions
type Manager struct {
	sessionDir     string
	currentSession *Session
	fs             filesystem.FileSystem
}

// NewManager creates a new history manager
func NewManager(sessionDir string) (*Manager, error) {
	return NewManagerWithFS(sessionDir, filesystem.NewOSFileSystem())
}

// NewManagerWithFS creates a new history manager with a custom FileSystem (for testing)
func NewManagerWithFS(sessionDir string, fs filesystem.FileSystem) (*Manager, error) {
	if err := fs.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	return &Manager{
		sessionDir: sessionDir,
		fs:         fs,
		currentSession: &Session{
			ID:        generateID(),
			Name:      "default",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Actions:   make([]Action, 0),
		},
	}, nil
}

// AddCommandAction adds a command action to history
func (m *Manager) AddCommandAction(command, output string) {
	action := Action{
		ID:        generateID(),
		Type:      ActionTypeCommand,
		Timestamp: time.Now(),
		Command:   command,
		Output:    output,
	}

	m.currentSession.Actions = append(m.currentSession.Actions, action)
	m.currentSession.UpdatedAt = time.Now()
}

// AddFileAction adds a file action to history
func (m *Manager) AddFileAction(filePath, oldValue, newValue string) {
	action := Action{
		ID:        generateID(),
		Type:      ActionTypeFile,
		Timestamp: time.Now(),
		FilePath:  filePath,
		OldValue:  oldValue,
		NewValue:  newValue,
	}

	m.currentSession.Actions = append(m.currentSession.Actions, action)
	m.currentSession.UpdatedAt = time.Now()
}

// GetActions returns all actions in current session
func (m *Manager) GetActions() []Action {
	return m.currentSession.Actions
}

// Rollback rolls back the last N actions
func (m *Manager) Rollback(count int) ([]Action, error) {
	if count <= 0 || count > len(m.currentSession.Actions) {
		return nil, fmt.Errorf("invalid rollback count")
	}

	// Get actions to rollback (in reverse order)
	actionsToRollback := make([]Action, count)
	copy(actionsToRollback, m.currentSession.Actions[len(m.currentSession.Actions)-count:])

	// Reverse to get chronological order
	for i, j := 0, len(actionsToRollback)-1; i < j; i, j = i+1, j-1 {
		actionsToRollback[i], actionsToRollback[j] = actionsToRollback[j], actionsToRollback[i]
	}

	// Remove from current session
	m.currentSession.Actions = m.currentSession.Actions[:len(m.currentSession.Actions)-count]
	m.currentSession.UpdatedAt = time.Now()

	return actionsToRollback, nil
}

// SaveSession saves the current session
func (m *Manager) SaveSession(name string) error {
	if name != "" {
		m.currentSession.Name = name
	}

	sessionFile := filepath.Join(m.sessionDir, fmt.Sprintf("%s.json", m.currentSession.ID))

	data, err := json.MarshalIndent(m.currentSession, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := m.fs.WriteFile(sessionFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// LoadSession loads a session by ID
func (m *Manager) LoadSession(sessionID string) error {
	sessionFile := filepath.Join(m.sessionDir, fmt.Sprintf("%s.json", sessionID))

	data, err := m.fs.ReadFile(sessionFile)
	if err != nil {
		return fmt.Errorf("failed to read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return fmt.Errorf("failed to unmarshal session: %w", err)
	}

	m.currentSession = &session
	return nil
}

// ListSessions lists all saved sessions
func (m *Manager) ListSessions() ([]Session, error) {
	files, err := m.fs.ReadDir(m.sessionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	var sessions []Session
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		data, err := m.fs.ReadFile(filepath.Join(m.sessionDir, file.Name()))
		if err != nil {
			continue
		}

		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// GetCurrentSession returns the current session
func (m *Manager) GetCurrentSession() *Session {
	return m.currentSession
}

// NewSession creates a new session
func (m *Manager) NewSession(name string) {
	m.currentSession = &Session{
		ID:        generateID(),
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Actions:   make([]Action, 0),
	}
}

// generateID generates a simple ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
