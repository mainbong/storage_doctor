package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewLogger(tmpDir, INFO)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}

	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}

	if logger.logDir != tmpDir {
		t.Errorf("Expected logDir '%s', got '%s'", tmpDir, logger.logDir)
	}

	if logger.level != INFO {
		t.Errorf("Expected level INFO, got %v", logger.level)
	}

	if logger.file == nil {
		t.Error("Expected file to be opened, got nil")
	}

	defer logger.Close()
}

func TestLogger_Levels(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewLogger(tmpDir, DEBUG)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer logger.Close()

	// All levels should be logged when level is DEBUG
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	// Verify log file was written
	logFiles, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read log directory: %v", err)
	}

	if len(logFiles) == 0 {
		t.Error("Expected log files to be created, got none")
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewLogger(tmpDir, WARN)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer logger.Close()

	// DEBUG and INFO should be filtered out
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	// Verify log file exists
	logFiles, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read log directory: %v", err)
	}

	if len(logFiles) == 0 {
		t.Error("Expected log files to be created, got none")
	}
}

func TestLogger_GetLogDir(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewLogger(tmpDir, INFO)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer logger.Close()

	if logger.GetLogDir() != tmpDir {
		t.Errorf("Expected logDir '%s', got '%s'", tmpDir, logger.GetLogDir())
	}
}

func TestLogger_Close(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewLogger(tmpDir, INFO)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}

	if logger.file == nil {
		t.Fatal("Expected file to be opened, got nil")
	}

	err = logger.Close()
	if err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// File should be nil after Close()
	if logger.file != nil {
		t.Error("Expected file to be nil after Close(), but it's not")
	}

	// Second Close() should not error
	err = logger.Close()
	if err != nil {
		t.Errorf("Second Close() should not error, got: %v", err)
	}
}

func TestLogger_LogMessageFormat(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewLogger(tmpDir, INFO)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer logger.Close()

	logger.Info("test message %d", 123)

	// Read log file and verify format
	logFiles, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read log directory: %v", err)
	}

	var logFile string
	for _, f := range logFiles {
		if strings.HasSuffix(f.Name(), ".log") && !strings.HasSuffix(f.Name(), "latest.log") {
			logFile = filepath.Join(tmpDir, f.Name())
			break
		}
	}

	if logFile == "" {
		t.Fatal("Could not find log file")
	}

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "INFO") {
		t.Error("Expected log to contain 'INFO'")
	}
	if !strings.Contains(logContent, "test message 123") {
		t.Error("Expected log to contain 'test message 123'")
	}
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()

	err := Init(tmpDir, INFO)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
	defer Close()

	if defaultLogger == nil {
		t.Fatal("Expected defaultLogger to be initialized, got nil")
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	tmpDir := t.TempDir()

	err := Init(tmpDir, INFO)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
	defer Close()

	// Test package-level functions
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")

	// Verify log file was written
	logFiles, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read log directory: %v", err)
	}

	if len(logFiles) == 0 {
		t.Error("Expected log files to be created, got none")
	}
}

func TestPackageLevelFunctions_NoInit(t *testing.T) {
	// Reset default logger
	defaultLogger = nil

	// These should not panic
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")
	Close()
}

func TestNewLogger_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new", "log", "dir")

	logger, err := NewLogger(newDir, INFO)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer logger.Close()

	// Verify directory was created
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("Expected log directory to be created, but it doesn't exist")
	}
}

func TestNewLogger_LogFileCreation(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewLogger(tmpDir, INFO)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer logger.Close()

	// Verify log file was created
	logFiles, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read log directory: %v", err)
	}

	foundLogFile := false
	for _, f := range logFiles {
		if strings.HasPrefix(f.Name(), "storage-doctor_") && strings.HasSuffix(f.Name(), ".log") {
			foundLogFile = true
			break
		}
	}

	if !foundLogFile {
		t.Error("Expected log file to be created, but it wasn't found")
	}
}

