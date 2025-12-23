package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mainbong/storage_doctor/internal/filesystem"
)

func TestNewManager(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	sessionDir := "/test/sessions"

	manager, err := NewManagerWithFS(sessionDir, mockFS)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.sessionDir != sessionDir {
		t.Errorf("Expected sessionDir '%s', got '%s'", sessionDir, manager.sessionDir)
	}

	if manager.currentSession == nil {
		t.Fatal("Expected currentSession to be initialized, got nil")
	}

	if manager.currentSession.Name != "default" {
		t.Errorf("Expected session name 'default', got '%s'", manager.currentSession.Name)
	}
}

func TestAddCommandAction(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager, _ := NewManagerWithFS("/test/sessions", mockFS)

	command := "kubectl get pods"
	output := "pod1 pod2 pod3"

	manager.AddCommandAction(command, output)

	actions := manager.GetActions()
	if len(actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(actions))
	}

	action := actions[0]
	if action.Type != ActionTypeCommand {
		t.Errorf("Expected ActionTypeCommand, got %v", action.Type)
	}
	if action.Command != command {
		t.Errorf("Expected command '%s', got '%s'", command, action.Command)
	}
	if action.Output != output {
		t.Errorf("Expected output '%s', got '%s'", output, action.Output)
	}
}

func TestAddFileAction(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager, _ := NewManagerWithFS("/test/sessions", mockFS)

	filePath := "/test/file.yaml"
	oldValue := "old content"
	newValue := "new content"

	manager.AddFileAction(filePath, oldValue, newValue)

	actions := manager.GetActions()
	if len(actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(actions))
	}

	action := actions[0]
	if action.Type != ActionTypeFile {
		t.Errorf("Expected ActionTypeFile, got %v", action.Type)
	}
	if action.FilePath != filePath {
		t.Errorf("Expected filePath '%s', got '%s'", filePath, action.FilePath)
	}
	if action.OldValue != oldValue {
		t.Errorf("Expected oldValue '%s', got '%s'", oldValue, action.OldValue)
	}
	if action.NewValue != newValue {
		t.Errorf("Expected newValue '%s', got '%s'", newValue, action.NewValue)
	}
}

func TestRollback(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager, _ := NewManagerWithFS("/test/sessions", mockFS)

	// Add multiple actions
	manager.AddCommandAction("cmd1", "output1")
	manager.AddCommandAction("cmd2", "output2")
	manager.AddFileAction("/test/file", "old", "new")

	actions := manager.GetActions()
	if len(actions) != 3 {
		t.Fatalf("Expected 3 actions, got %d", len(actions))
	}

	// Rollback last 2 actions
	rolledBack, err := manager.Rollback(2)
	if err != nil {
		t.Fatalf("Rollback() failed: %v", err)
	}

	if len(rolledBack) != 2 {
		t.Errorf("Expected 2 rolled back actions, got %d", len(rolledBack))
	}

	// Verify actions were removed
	remainingActions := manager.GetActions()
	if len(remainingActions) != 1 {
		t.Errorf("Expected 1 remaining action, got %d", len(remainingActions))
	}
}

func TestRollback_InvalidCount(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager, _ := NewManagerWithFS("/test/sessions", mockFS)

	manager.AddCommandAction("cmd1", "output1")

	// Try to rollback more than available
	_, err := manager.Rollback(10)
	if err == nil {
		t.Error("Expected error for invalid rollback count, got nil")
	}

	// Try to rollback zero
	_, err = manager.Rollback(0)
	if err == nil {
		t.Error("Expected error for zero rollback count, got nil")
	}
}

func TestSaveSession(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	sessionDir := "/test/sessions"
	manager, _ := NewManagerWithFS(sessionDir, mockFS)

	manager.AddCommandAction("test command", "test output")

	err := manager.SaveSession("test-session")
	if err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	// Verify session was saved
	sessionFile := filepath.Join(sessionDir, manager.currentSession.ID+".json")
	savedData := mockFS.GetFile(sessionFile)
	if len(savedData) == 0 {
		t.Error("Expected session to be saved, but file is empty")
	}

	// Verify JSON is valid
	var session Session
	if err := json.Unmarshal(savedData, &session); err != nil {
		t.Fatalf("Saved session is not valid JSON: %v", err)
	}

	if session.Name != "test-session" {
		t.Errorf("Expected session name 'test-session', got '%s'", session.Name)
	}
	if len(session.Actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(session.Actions))
	}
}

func TestSaveSession_EmptyName(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager, _ := NewManagerWithFS("/test/sessions", mockFS)

	// Save with empty name (should use current name)
	originalName := manager.currentSession.Name
	err := manager.SaveSession("")
	if err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	if manager.currentSession.Name != originalName {
		t.Errorf("Expected session name to remain '%s', got '%s'", originalName, manager.currentSession.Name)
	}
}

func TestLoadSession(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	sessionDir := "/test/sessions"
	manager, _ := NewManagerWithFS(sessionDir, mockFS)

	// Create a session to save
	manager.AddCommandAction("cmd1", "output1")
	manager.AddFileAction("/test/file", "old", "new")
	sessionID := manager.currentSession.ID
	err := manager.SaveSession("test-session")
	if err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	// Create new manager and load session
	newManager, _ := NewManagerWithFS(sessionDir, mockFS)
	err = newManager.LoadSession(sessionID)
	if err != nil {
		t.Fatalf("LoadSession() failed: %v", err)
	}

	// Verify session was loaded
	if newManager.currentSession.ID != sessionID {
		t.Errorf("Expected session ID '%s', got '%s'", sessionID, newManager.currentSession.ID)
	}
	if newManager.currentSession.Name != "test-session" {
		t.Errorf("Expected session name 'test-session', got '%s'", newManager.currentSession.Name)
	}
	if len(newManager.currentSession.Actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(newManager.currentSession.Actions))
	}
}

func TestLoadSession_NotFound(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager, _ := NewManagerWithFS("/test/sessions", mockFS)

	err := manager.LoadSession("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent session, got nil")
	}
}

func TestLoadSession_InvalidJSON(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	sessionDir := "/test/sessions"
	sessionID := "test-id"
	sessionFile := filepath.Join(sessionDir, sessionID+".json")

	invalidJSON := []byte("{ invalid json }")
	mockFS.AddFile(sessionFile, invalidJSON, 0644)
	mockFS.AddDir(sessionDir, 0755)

	manager, _ := NewManagerWithFS(sessionDir, mockFS)
	err := manager.LoadSession(sessionID)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestListSessions(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	sessionDir := "/test/sessions"
	manager, _ := NewManagerWithFS(sessionDir, mockFS)

	// Create and save multiple sessions
	manager.AddCommandAction("cmd1", "output1")
	manager.SaveSession("session1")
	session1ID := manager.currentSession.ID

	manager.NewSession("session2")
	manager.AddFileAction("/test/file", "old", "new")
	manager.SaveSession("session2")
	session2ID := manager.currentSession.ID

	// List sessions
	sessions, err := manager.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() failed: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}

	// Verify session IDs
	sessionIDs := make(map[string]bool)
	for _, s := range sessions {
		sessionIDs[s.ID] = true
	}
	if !sessionIDs[session1ID] {
		t.Error("Expected session1 to be in list")
	}
	if !sessionIDs[session2ID] {
		t.Error("Expected session2 to be in list")
	}
}

func TestListSessions_Empty(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager, _ := NewManagerWithFS("/test/sessions", mockFS)

	sessions, err := manager.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() failed: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}
}

func TestNewSession(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager, _ := NewManagerWithFS("/test/sessions", mockFS)

	originalID := manager.currentSession.ID
	manager.AddCommandAction("cmd1", "output1")

	// Small delay to ensure different timestamp
	time.Sleep(time.Millisecond)
	manager.NewSession("new-session")

	if manager.currentSession.ID == originalID {
		t.Error("Expected new session to have different ID")
	}
	if manager.currentSession.Name != "new-session" {
		t.Errorf("Expected session name 'new-session', got '%s'", manager.currentSession.Name)
	}
	if len(manager.currentSession.Actions) != 0 {
		t.Errorf("Expected 0 actions in new session, got %d", len(manager.currentSession.Actions))
	}
}

func TestGetCurrentSession(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager, _ := NewManagerWithFS("/test/sessions", mockFS)

	session := manager.GetCurrentSession()
	if session == nil {
		t.Fatal("Expected current session, got nil")
	}

	if session.ID != manager.currentSession.ID {
		t.Error("Expected GetCurrentSession() to return current session")
	}
}

func TestSaveSession_WriteError(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	sessionDir := "/test/sessions"
	manager, _ := NewManagerWithFS(sessionDir, mockFS)

	sessionFile := filepath.Join(sessionDir, manager.currentSession.ID+".json")
	mockFS.SetWriteError(sessionFile, os.ErrPermission)

	err := manager.SaveSession("test")
	if err == nil {
		t.Error("Expected error for write failure, got nil")
	}
}

