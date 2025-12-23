package files

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mainbong/storage_doctor/internal/filesystem"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Manager handles file operations
type Manager struct {
	backupDir string
	fs        filesystem.FileSystem
}

// NewManager creates a new file manager
func NewManager(backupDir string) *Manager {
	return NewManagerWithFS(backupDir, filesystem.NewOSFileSystem())
}

// NewManagerWithFS creates a new file manager with a custom FileSystem (for testing)
func NewManagerWithFS(backupDir string, fs filesystem.FileSystem) *Manager {
	return &Manager{
		backupDir: backupDir,
		fs:        fs,
	}
}

// ReadFile reads a file and returns its contents
func (m *Manager) ReadFile(path string) (string, error) {
	data, err := m.fs.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(data), nil
}

// WriteFile writes content to a file
func (m *Manager) WriteFile(path, content string) error {
	// Create backup before writing
	if err := m.backupFile(path); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := m.fs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := m.fs.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ParseYAML parses a YAML file
func (m *Manager) ParseYAML(path string) (interface{}, error) {
	data, err := m.fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var result interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return result, nil
}

// WriteYAML writes data as YAML to a file
func (m *Manager) WriteYAML(path string, data interface{}) error {
	// Create backup
	if err := m.backupFile(path); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := m.fs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := m.fs.WriteFile(path, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ParseJSON parses a JSON file
func (m *Manager) ParseJSON(path string) (interface{}, error) {
	data, err := m.fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return result, nil
}

// WriteJSON writes data as JSON to a file
func (m *Manager) WriteJSON(path string, data interface{}) error {
	// Create backup
	if err := m.backupFile(path); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := m.fs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := m.fs.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ParseTOML parses a TOML file
func (m *Manager) ParseTOML(path string) (interface{}, error) {
	data, err := m.fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var result interface{}
	if err := toml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	return result, nil
}

// WriteTOML writes data as TOML to a file
func (m *Manager) WriteTOML(path string, data interface{}) error {
	// Create backup
	if err := m.backupFile(path); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	tomlData, err := toml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal TOML: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := m.fs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := m.fs.WriteFile(path, tomlData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// SearchInFile searches for a pattern in a file
func (m *Manager) SearchInFile(path, pattern string) ([]string, error) {
	content, err := m.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(content, "\n")
	var matches []string

	for i, line := range lines {
		if strings.Contains(line, pattern) {
			matches = append(matches, fmt.Sprintf("%d: %s", i+1, line))
		}
	}

	return matches, nil
}

// backupFile creates a backup of a file
func (m *Manager) backupFile(path string) error {
	if m.backupDir == "" {
		return nil // No backup directory specified
	}

	// Check if file exists
	if _, err := m.fs.Stat(path); os.IsNotExist(err) {
		return nil // File doesn't exist, no need to backup
	}

	// Create backup directory
	if err := m.fs.MkdirAll(m.backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename
	baseName := filepath.Base(path)
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(path))
	pathHash := fmt.Sprintf("%08x", hasher.Sum32())
	timestamp := time.Now().UTC().Format("20060102-150405.000000000")
	backupPath := filepath.Join(m.backupDir, fmt.Sprintf("%s.%s.%s.backup", baseName, timestamp, pathHash))

	// Read original file
	data, err := m.fs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file for backup: %w", err)
	}

	// Write backup
	if err := m.fs.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return nil
}

// GetFileType determines the file type based on extension
func (m *Manager) GetFileType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".toml":
		return "toml"
	case ".ini", ".cfg", ".conf":
		return "ini"
	default:
		return "text"
	}
}
