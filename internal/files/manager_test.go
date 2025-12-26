package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mainbong/storage_doctor/internal/filesystem"
)

func TestReadFile(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager := NewManagerWithFS("/test/backups", mockFS)

	testFile := "/test/file.txt"
	testContent := "test content"

	mockFS.AddFile(testFile, []byte(testContent), 0644)

	content, err := manager.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if content != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, content)
	}
}

func TestReadFile_NotFound(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager := NewManagerWithFS("/test/backups", mockFS)

	_, err := manager.ReadFile("/nonexistent.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestWriteFile(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	backupDir := "/test/backups"
	manager := NewManagerWithFS(backupDir, mockFS)

	testFile := "/test/file.txt"
	testContent := "new content"

	// Create existing file for backup
	mockFS.AddFile(testFile, []byte("old content"), 0644)
	mockFS.AddDir("/test", 0755)
	mockFS.AddDir(backupDir, 0755)

	err := manager.WriteFile(testFile, testContent)
	if err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	// Verify file was written
	content := string(mockFS.GetFile(testFile))
	if content != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, content)
	}

	// Verify backup was created
	entries, err := mockFS.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("ReadDir() failed: %v", err)
	}

	var backupName string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "file.txt.") && strings.HasSuffix(name, ".backup") {
			backupName = name
			break
		}
	}
	if backupName == "" {
		t.Fatalf("Expected backup file for file.txt, got none")
	}

	backupFile := filepath.Join(backupDir, backupName)
	backupContent := string(mockFS.GetFile(backupFile))
	if backupContent != "old content" {
		t.Errorf("Expected backup content 'old content', got '%s'", backupContent)
	}
}

func TestWriteFile_NoBackupDir(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager := NewManagerWithFS("", mockFS) // No backup dir

	testFile := "/test/file.txt"
	testContent := "new content"

	mockFS.AddDir("/test", 0755)

	err := manager.WriteFile(testFile, testContent)
	if err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	// Verify file was written
	content := string(mockFS.GetFile(testFile))
	if content != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, content)
	}
}

func TestWriteFile_WriteError(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager := NewManagerWithFS("/test/backups", mockFS)

	testFile := "/test/file.txt"
	mockFS.AddFile(testFile, []byte("old"), 0644)
	mockFS.AddDir("/test", 0755)
	mockFS.SetWriteError(testFile, os.ErrPermission)

	err := manager.WriteFile(testFile, "new")
	if err == nil {
		t.Error("Expected error for write failure, got nil")
	}
}

func TestParseYAML(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager := NewManagerWithFS("/test/backups", mockFS)

	testFile := "/test/config.yaml"
	yamlContent := `name: test
value: 42
nested:
  key: value
`

	mockFS.AddFile(testFile, []byte(yamlContent), 0644)

	result, err := manager.ParseYAML(testFile)
	if err != nil {
		t.Fatalf("ParseYAML() failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", result)
	}

	if resultMap["name"] != "test" {
		t.Errorf("Expected name 'test', got '%v'", resultMap["name"])
	}
}

func TestParseYAML_Invalid(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager := NewManagerWithFS("/test/backups", mockFS)

	testFile := "/test/invalid.yaml"
	invalidYAML := "invalid: yaml: content: ["

	mockFS.AddFile(testFile, []byte(invalidYAML), 0644)

	_, err := manager.ParseYAML(testFile)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestWriteYAML(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager := NewManagerWithFS("/test/backups", mockFS)

	testFile := "/test/config.yaml"
	data := map[string]interface{}{
		"name":  "test",
		"value": 42,
	}

	mockFS.AddDir("/test", 0755)

	err := manager.WriteYAML(testFile, data)
	if err != nil {
		t.Fatalf("WriteYAML() failed: %v", err)
	}

	// Verify file was written
	content := string(mockFS.GetFile(testFile))
	if len(content) == 0 {
		t.Error("Expected YAML content, got empty")
	}
}

func TestParseJSON(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager := NewManagerWithFS("/test/backups", mockFS)

	testFile := "/test/config.json"
	jsonContent := `{"name":"test","value":42}`

	mockFS.AddFile(testFile, []byte(jsonContent), 0644)

	result, err := manager.ParseJSON(testFile)
	if err != nil {
		t.Fatalf("ParseJSON() failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", result)
	}

	if resultMap["name"] != "test" {
		t.Errorf("Expected name 'test', got '%v'", resultMap["name"])
	}
}

func TestParseJSON_Invalid(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager := NewManagerWithFS("/test/backups", mockFS)

	testFile := "/test/invalid.json"
	invalidJSON := "{ invalid json }"

	mockFS.AddFile(testFile, []byte(invalidJSON), 0644)

	_, err := manager.ParseJSON(testFile)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestWriteJSON(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager := NewManagerWithFS("/test/backups", mockFS)

	testFile := "/test/config.json"
	data := map[string]interface{}{
		"name":  "test",
		"value": 42,
	}

	mockFS.AddDir("/test", 0755)

	err := manager.WriteJSON(testFile, data)
	if err != nil {
		t.Fatalf("WriteJSON() failed: %v", err)
	}

	// Verify file was written
	content := string(mockFS.GetFile(testFile))
	if len(content) == 0 {
		t.Error("Expected JSON content, got empty")
	}
}

func TestParseTOML(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager := NewManagerWithFS("/test/backups", mockFS)

	testFile := "/test/config.toml"
	tomlContent := `name = "test"
value = 42
`

	mockFS.AddFile(testFile, []byte(tomlContent), 0644)

	result, err := manager.ParseTOML(testFile)
	if err != nil {
		t.Fatalf("ParseTOML() failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", result)
	}

	if resultMap["name"] != "test" {
		t.Errorf("Expected name 'test', got '%v'", resultMap["name"])
	}
}

func TestWriteTOML(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager := NewManagerWithFS("/test/backups", mockFS)

	testFile := "/test/config.toml"
	data := map[string]interface{}{
		"name":  "test",
		"value": 42,
	}

	mockFS.AddDir("/test", 0755)

	err := manager.WriteTOML(testFile, data)
	if err != nil {
		t.Fatalf("WriteTOML() failed: %v", err)
	}

	// Verify file was written
	content := string(mockFS.GetFile(testFile))
	if len(content) == 0 {
		t.Error("Expected TOML content, got empty")
	}
}

func TestSearchInFile(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager := NewManagerWithFS("/test/backups", mockFS)

	testFile := "/test/file.txt"
	content := `line 1: test
line 2: example
line 3: test again
line 4: other
`

	mockFS.AddFile(testFile, []byte(content), 0644)

	matches, err := manager.SearchInFile(testFile, "test")
	if err != nil {
		t.Fatalf("SearchInFile() failed: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}
}

func TestGetFileType(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	manager := NewManagerWithFS("/test/backups", mockFS)

	tests := []struct {
		path     string
		expected string
	}{
		{"/test/file.yaml", "yaml"},
		{"/test/file.yml", "yaml"},
		{"/test/file.json", "json"},
		{"/test/file.toml", "toml"},
		{"/test/file.ini", "ini"},
		{"/test/file.cfg", "ini"},
		{"/test/file.conf", "ini"},
		{"/test/file.txt", "text"},
		{"/test/file", "text"},
	}

	for _, tt := range tests {
		result := manager.GetFileType(tt.path)
		if result != tt.expected {
			t.Errorf("GetFileType(%s) = %s, expected %s", tt.path, result, tt.expected)
		}
	}
}




