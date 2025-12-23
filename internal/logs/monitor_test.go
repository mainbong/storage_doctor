package logs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewMonitor(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.log")

	monitor, err := NewMonitor(tmpFile)
	if err != nil {
		t.Fatalf("NewMonitor() failed: %v", err)
	}

	if monitor == nil {
		t.Fatal("NewMonitor() returned nil")
	}

	if monitor.filePath != tmpFile {
		t.Errorf("Expected filePath '%s', got '%s'", tmpFile, monitor.filePath)
	}

	if monitor.watcher == nil {
		t.Error("Expected watcher to be initialized, got nil")
	}
}

func TestSearch(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.log")

	// Create test log file
	logContent := `2025-01-01 10:00:00 INFO: Application started
2025-01-01 10:01:00 ERROR: Database connection failed
2025-01-01 10:02:00 WARN: High memory usage
2025-01-01 10:03:00 ERROR: Failed to process request
2025-01-01 10:04:00 INFO: Request completed
`

	if err := os.WriteFile(tmpFile, []byte(logContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	monitor, err := NewMonitor(tmpFile)
	if err != nil {
		t.Fatalf("NewMonitor() failed: %v", err)
	}

	// Search for ERROR lines
	matches, err := monitor.Search("ERROR")
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}

	// Verify matches contain ERROR
	for _, match := range matches {
		if !strings.Contains(match, "ERROR") {
			t.Errorf("Expected match to contain 'ERROR', got '%s'", match)
		}
	}
}

func TestSearch_InvalidPattern(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.log")
	os.WriteFile(tmpFile, []byte("test"), 0644)

	monitor, err := NewMonitor(tmpFile)
	if err != nil {
		t.Fatalf("NewMonitor() failed: %v", err)
	}

	_, err = monitor.Search("[invalid regex")
	if err == nil {
		t.Error("Expected error for invalid regex pattern, got nil")
	}
}

func TestFilter(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.log")

	// Create test log file
	logContent := `2025-01-01 10:00:00 INFO: Application started
2025-01-01 10:01:00 ERROR: Database connection failed
2025-01-01 10:02:00 WARN: High memory usage
2025-01-01 10:03:00 ERROR: Failed to process request
2025-01-01 10:04:00 INFO: Request completed
`

	if err := os.WriteFile(tmpFile, []byte(logContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	monitor, err := NewMonitor(tmpFile)
	if err != nil {
		t.Fatalf("NewMonitor() failed: %v", err)
	}

	// Filter ERROR level
	matches, err := monitor.Filter("ERROR")
	if err != nil {
		t.Fatalf("Filter() failed: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Expected 2 ERROR matches, got %d", len(matches))
	}

	// Filter WARN level
	matches, err = monitor.Filter("WARN")
	if err != nil {
		t.Fatalf("Filter() failed: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("Expected 1 WARN match, got %d", len(matches))
	}
}

func TestSummarize(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.log")

	// Create test log file
	logContent := `2025-01-01 10:00:00 INFO: Application started
2025-01-01 10:01:00 ERROR: Database connection failed
2025-01-01 10:02:00 WARN: High memory usage
2025-01-01 10:03:00 ERROR: Failed to process request
2025-01-01 10:04:00 INFO: Request completed
`

	if err := os.WriteFile(tmpFile, []byte(logContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	monitor, err := NewMonitor(tmpFile)
	if err != nil {
		t.Fatalf("NewMonitor() failed: %v", err)
	}

	summary, err := monitor.Summarize()
	if err != nil {
		t.Fatalf("Summarize() failed: %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary, got nil")
	}

	// Verify summary contains expected fields
	if _, ok := summary["total_lines"]; !ok {
		t.Error("Expected summary to contain 'total_lines'")
	}
	if _, ok := summary["error_count"]; !ok {
		t.Error("Expected summary to contain 'error_count'")
	}
	if _, ok := summary["warn_count"]; !ok {
		t.Error("Expected summary to contain 'warn_count'")
	}
	if _, ok := summary["info_count"]; !ok {
		t.Error("Expected summary to contain 'info_count'")
	}

	// Verify counts
	if summary["error_count"].(int) != 2 {
		t.Errorf("Expected 2 errors, got %v", summary["error_count"])
	}
	if summary["warn_count"].(int) != 1 {
		t.Errorf("Expected 1 warning, got %v", summary["warn_count"])
	}
	if summary["info_count"].(int) != 2 {
		t.Errorf("Expected 2 info messages, got %v", summary["info_count"])
	}
}

func TestTail_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.log")

	os.WriteFile(tmpFile, []byte("initial line\n"), 0644)

	monitor, err := NewMonitor(tmpFile)
	if err != nil {
		t.Fatalf("NewMonitor() failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start tailing in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- monitor.Tail(ctx, func(line string) {
			// Do nothing
		})
	}()

	// Cancel context after short delay
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Wait for tail to finish
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Tail() did not return after context cancellation")
	}
}

func TestSearch_FileNotFound(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "nonexistent.log")

	monitor, err := NewMonitor(tmpFile)
	if err != nil {
		t.Fatalf("NewMonitor() failed: %v", err)
	}

	_, err = monitor.Search("pattern")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestFilter_FileNotFound(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "nonexistent.log")

	monitor, err := NewMonitor(tmpFile)
	if err != nil {
		t.Fatalf("NewMonitor() failed: %v", err)
	}

	_, err = monitor.Filter("ERROR")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestSummarize_FileNotFound(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "nonexistent.log")

	monitor, err := NewMonitor(tmpFile)
	if err != nil {
		t.Fatalf("NewMonitor() failed: %v", err)
	}

	_, err = monitor.Summarize()
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

