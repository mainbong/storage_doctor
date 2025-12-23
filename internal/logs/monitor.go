package logs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Monitor monitors log files
type Monitor struct {
	filePath    string
	watcher     *fsnotify.Watcher
	lastOffset  int64
	lastPartial string
}

// NewMonitor creates a new log monitor
func NewMonitor(filePath string) (*Monitor, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	return &Monitor{
		filePath: filePath,
		watcher:  watcher,
	}, nil
}

// Tail tails a log file and calls onLine for each new line
func (m *Monitor) Tail(ctx context.Context, onLine func(string)) error {
	// Add file to watcher
	if err := m.watcher.Add(m.filePath); err != nil {
		return fmt.Errorf("failed to add file to watcher: %w", err)
	}

	// Read existing content first (last 100 lines)
	if err := m.readLastLines(100, onLine); err != nil {
		return fmt.Errorf("failed to read last lines: %w", err)
	}
	if err := m.setInitialOffset(); err != nil {
		return fmt.Errorf("failed to read initial offset: %w", err)
	}

	// Watch for new lines
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-m.watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				// Read new lines
				if err := m.readNewLines(onLine); err != nil {
					return fmt.Errorf("failed to read new lines: %w", err)
				}
			}
		case err := <-m.watcher.Errors:
			return fmt.Errorf("watcher error: %w", err)
		}
	}
}

// Search searches for a pattern in the log file
func (m *Monitor) Search(pattern string) ([]string, error) {
	file, err := os.Open(m.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var matches []string
	scanner := bufio.NewScanner(file)
	lineNum := 1

	for scanner.Scan() {
		line := scanner.Text()
		if re.MatchString(line) {
			matches = append(matches, fmt.Sprintf("%d: %s", lineNum, line))
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}

	return matches, nil
}

// Filter filters log lines by level (ERROR, WARN, INFO, etc.)
func (m *Monitor) Filter(level string) ([]string, error) {
	file, err := os.Open(m.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var matches []string
	scanner := bufio.NewScanner(file)
	levelUpper := strings.ToUpper(level)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.ToUpper(line), levelUpper) {
			matches = append(matches, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}

	return matches, nil
}

// Summarize summarizes log file statistics
func (m *Monitor) Summarize() (map[string]interface{}, error) {
	file, err := os.Open(m.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	totalLines := 0
	errorCount := 0
	warnCount := 0
	infoCount := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		totalLines++
		line := strings.ToUpper(scanner.Text())
		if strings.Contains(line, "ERROR") {
			errorCount++
		}
		if strings.Contains(line, "WARN") {
			warnCount++
		}
		if strings.Contains(line, "INFO") {
			infoCount++
		}
	}

	stats := map[string]interface{}{
		"total_lines": totalLines,
		"error_count": errorCount,
		"warn_count":  warnCount,
		"info_count":  infoCount,
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}

	return stats, nil
}

// readLastLines reads the last N lines from the file
func (m *Monitor) readLastLines(n int, onLine func(string)) error {
	file, err := os.Open(m.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan file: %w", err)
	}

	// Get last N lines
	start := len(lines) - n
	if start < 0 {
		start = 0
	}

	for i := start; i < len(lines); i++ {
		onLine(lines[i])
	}

	return nil
}

// readNewLines reads new lines since last read
func (m *Monitor) readNewLines(onLine func(string)) error {
	file, err := os.Open(m.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if stat.Size() < m.lastOffset {
		m.lastOffset = 0
		m.lastPartial = ""
	}

	if _, err := file.Seek(m.lastOffset, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read new data: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	m.lastOffset = stat.Size()

	text := m.lastPartial + string(data)
	if !strings.HasSuffix(text, "\n") {
		lastNewline := strings.LastIndex(text, "\n")
		if lastNewline == -1 {
			m.lastPartial = text
			return nil
		}
		m.lastPartial = text[lastNewline+1:]
		text = text[:lastNewline+1]
	} else {
		m.lastPartial = ""
	}

	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	for _, line := range lines {
		onLine(line)
	}

	return nil
}

// Close closes the monitor
func (m *Monitor) Close() error {
	if m.watcher != nil {
		return m.watcher.Close()
	}
	return nil
}

// TailWithTimeout tails a log file with a timeout
func (m *Monitor) TailWithTimeout(timeout time.Duration, onLine func(string)) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return m.Tail(ctx, onLine)
}

func (m *Monitor) setInitialOffset() error {
	file, err := os.Open(m.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	m.lastOffset = stat.Size()
	m.lastPartial = ""
	return nil
}
