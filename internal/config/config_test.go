package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mainbong/storage_doctor/internal/filesystem"
)

func TestLoad_DefaultConfig(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	testDir := "/test/config"
	testFile := filepath.Join(testDir, "config.json")

	cfg, err := LoadWithFS(mockFS, testDir, testFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check defaults
	if cfg.LLMProvider != "anthropic" {
		t.Errorf("Expected LLMProvider 'anthropic', got '%s'", cfg.LLMProvider)
	}
	if cfg.Anthropic.Model != "claude-haiku-4-5-20251001" {
		t.Errorf("Expected Anthropic model 'claude-haiku-4-5-20251001', got '%s'", cfg.Anthropic.Model)
	}
	if cfg.OpenAI.Model != "gpt-5" {
		t.Errorf("Expected OpenAI model 'gpt-5', got '%s'", cfg.OpenAI.Model)
	}
	if cfg.Search.Provider != "duckduckgo" {
		t.Errorf("Expected Search provider 'duckduckgo', got '%s'", cfg.Search.Provider)
	}
	if cfg.AutoApproveCommands != false {
		t.Errorf("Expected AutoApproveCommands false, got %v", cfg.AutoApproveCommands)
	}

	// Check that default config was saved
	savedData := mockFS.GetFile(testFile)
	if len(savedData) == 0 {
		t.Error("Expected default config to be saved, but file is empty")
	}
}

func TestLoad_DefaultConfig_UsesEnvKeys(t *testing.T) {
	os.Setenv("ANTHROPIC_API_KEY", "env-anthropic-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	mockFS := filesystem.NewMockFileSystem()
	testDir := "/test/config"
	testFile := filepath.Join(testDir, "config.json")

	cfg, err := LoadWithFS(mockFS, testDir, testFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Anthropic.APIKey != "env-anthropic-key" {
		t.Errorf("Expected Anthropic API key from env, got '%s'", cfg.Anthropic.APIKey)
	}

	savedData := mockFS.GetFile(testFile)
	if len(savedData) == 0 {
		t.Fatal("Expected config to be saved")
	}

	var saved Config
	if err := json.Unmarshal(savedData, &saved); err != nil {
		t.Fatalf("Saved config is not valid JSON: %v", err)
	}
	if saved.Anthropic.APIKey != "env-anthropic-key" {
		t.Errorf("Expected saved Anthropic API key from env, got '%s'", saved.Anthropic.APIKey)
	}
}

func TestLoad_ExistingConfig(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	testDir := "/test/config"
	testFile := filepath.Join(testDir, "config.json")

	// Create existing config
	existingConfig := &Config{
		LLMProvider: "openai",
		OpenAI: struct {
			APIKey string `json:"api_key"`
			Model  string `json:"model"`
		}{
			Model: "gpt-4",
		},
		Search: struct {
			Provider string `json:"provider"`
			Google   struct {
				APIKey string `json:"api_key"`
				CX     string `json:"cx"`
			} `json:"google"`
			Bing struct {
				APIKey string `json:"api_key"`
			} `json:"bing"`
			Serper struct {
				APIKey string `json:"api_key"`
			} `json:"serper"`
		}{
			Provider: "google",
		},
		AutoApproveCommands: true,
		SessionDir:          "/custom/sessions",
		BackupDir:           "/custom/backups",
		LogDir:              "/custom/logs",
		LogLevel:            "debug",
	}

	data, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	mockFS.AddFile(testFile, data, 0600)
	mockFS.AddDir(testDir, 0755)

	cfg, err := LoadWithFS(mockFS, testDir, testFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check loaded values
	if cfg.LLMProvider != "openai" {
		t.Errorf("Expected LLMProvider 'openai', got '%s'", cfg.LLMProvider)
	}
	if cfg.OpenAI.Model != "gpt-4" {
		t.Errorf("Expected OpenAI model 'gpt-4', got '%s'", cfg.OpenAI.Model)
	}
	if cfg.Search.Provider != "google" {
		t.Errorf("Expected Search provider 'google', got '%s'", cfg.Search.Provider)
	}
	if cfg.AutoApproveCommands != true {
		t.Errorf("Expected AutoApproveCommands true, got %v", cfg.AutoApproveCommands)
	}
	if cfg.SessionDir != "/custom/sessions" {
		t.Errorf("Expected SessionDir '/custom/sessions', got '%s'", cfg.SessionDir)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LogLevel 'debug', got '%s'", cfg.LogLevel)
	}
}

func TestLoad_EmptyDirsFallbackToDefaults(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	testDir := "/test/config"
	testFile := filepath.Join(testDir, "config.json")

	existingConfig := &Config{
		LLMProvider: "anthropic",
		SessionDir:  "",
		BackupDir:   "",
		LogDir:      "",
	}

	data, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	mockFS.AddFile(testFile, data, 0600)
	mockFS.AddDir(testDir, 0755)

	cfg, err := LoadWithFS(mockFS, testDir, testFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.SessionDir != filepath.Join(testDir, "sessions") {
		t.Errorf("Expected SessionDir '%s', got '%s'", filepath.Join(testDir, "sessions"), cfg.SessionDir)
	}
	if cfg.BackupDir != filepath.Join(testDir, "backups") {
		t.Errorf("Expected BackupDir '%s', got '%s'", filepath.Join(testDir, "backups"), cfg.BackupDir)
	}
	if cfg.LogDir != filepath.Join(testDir, "logs") {
		t.Errorf("Expected LogDir '%s', got '%s'", filepath.Join(testDir, "logs"), cfg.LogDir)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	testDir := "/test/config"
	testFile := filepath.Join(testDir, "config.json")

	// Create invalid JSON
	invalidJSON := []byte("{ invalid json }")
	mockFS.AddFile(testFile, invalidJSON, 0600)
	mockFS.AddDir(testDir, 0755)

	_, err := LoadWithFS(mockFS, testDir, testFile)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestLoad_ReadError(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	testDir := "/test/config"
	testFile := filepath.Join(testDir, "config.json")

	// File exists but read fails
	mockFS.AddDir(testDir, 0755)
	mockFS.AddFile(testFile, []byte("{}"), 0600) // File exists
	mockFS.SetReadError(testFile, os.ErrPermission) // But read fails

	_, err := LoadWithFS(mockFS, testDir, testFile)
	if err == nil {
		t.Error("Expected error for read failure, got nil")
	}
}

func TestSave(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	testFile := "/test/config.json"

	cfg := &Config{
		LLMProvider: "anthropic",
		Anthropic: struct {
			APIKey string `json:"api_key"`
			Model  string `json:"model"`
		}{
			Model: "claude-haiku-4-5-20251001",
		},
		OpenAI: struct {
			APIKey string `json:"api_key"`
			Model  string `json:"model"`
		}{
			Model: "gpt-5",
		},
		Search: struct {
			Provider string `json:"provider"`
			Google   struct {
				APIKey string `json:"api_key"`
				CX     string `json:"cx"`
			} `json:"google"`
			Bing struct {
				APIKey string `json:"api_key"`
			} `json:"bing"`
			Serper struct {
				APIKey string `json:"api_key"`
			} `json:"serper"`
		}{
			Provider: "duckduckgo",
		},
		AutoApproveCommands: false,
		SessionDir:          "/test/sessions",
		BackupDir:           "/test/backups",
		LogDir:              "/test/logs",
		LogLevel:            "info",
	}

	err := cfg.SaveWithFS(mockFS, testFile)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file was written
	savedData := mockFS.GetFile(testFile)
	if len(savedData) == 0 {
		t.Error("Expected config to be saved, but file is empty")
	}

	// Verify JSON is valid
	var loadedConfig Config
	if err := json.Unmarshal(savedData, &loadedConfig); err != nil {
		t.Fatalf("Saved config is not valid JSON: %v", err)
	}

	// Verify values
	if loadedConfig.LLMProvider != cfg.LLMProvider {
		t.Errorf("Expected LLMProvider '%s', got '%s'", cfg.LLMProvider, loadedConfig.LLMProvider)
	}
}

func TestSave_WriteError(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	testFile := "/test/config.json"

	mockFS.SetWriteError(testFile, os.ErrPermission)

	cfg := &Config{}
	err := cfg.SaveWithFS(mockFS, testFile)
	if err == nil {
		t.Error("Expected error for write failure, got nil")
	}
}

func TestLoadAPIKeys_FromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("ANTHROPIC_API_KEY", "test-anthropic-key")
	os.Setenv("OPENAI_API_KEY", "test-openai-key")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("OPENAI_API_KEY")
	}()

	mockFS := filesystem.NewMockFileSystem()
	testDir := "/test/config"
	testFile := filepath.Join(testDir, "config.json")

	cfg, err := LoadWithFS(mockFS, testDir, testFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check that API keys were loaded from environment
	if cfg.Anthropic.APIKey != "test-anthropic-key" {
		t.Errorf("Expected Anthropic API key from env, got '%s'", cfg.Anthropic.APIKey)
	}
	if cfg.OpenAI.APIKey != "test-openai-key" {
		t.Errorf("Expected OpenAI API key from env, got '%s'", cfg.OpenAI.APIKey)
	}
}

func TestSet_ConfigValues(t *testing.T) {
	cfg := &Config{}

	if err := cfg.Set("llm_provider", "openai"); err != nil {
		t.Fatalf("Set() failed: %v", err)
	}
	if cfg.LLMProvider != "openai" {
		t.Errorf("Expected LLMProvider 'openai', got '%s'", cfg.LLMProvider)
	}

	if err := cfg.Set("auto_approve_commands", "true"); err != nil {
		t.Fatalf("Set() failed: %v", err)
	}
	if cfg.AutoApproveCommands != true {
		t.Errorf("Expected AutoApproveCommands true, got %v", cfg.AutoApproveCommands)
	}

	if err := cfg.Set("search.provider", "duckduckgo"); err != nil {
		t.Fatalf("Set() failed: %v", err)
	}
	if cfg.Search.Provider != "duckduckgo" {
		t.Errorf("Expected Search provider 'duckduckgo', got '%s'", cfg.Search.Provider)
	}
}

func TestSet_InvalidKey(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Set("unknown.key", "value"); err == nil {
		t.Error("Expected error for unknown key, got nil")
	}
}

func TestSet_InvalidValue(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Set("llm_provider", "invalid"); err == nil {
		t.Error("Expected error for invalid llm_provider, got nil")
	}
	if err := cfg.Set("auto_approve_commands", "notabool"); err == nil {
		t.Error("Expected error for invalid auto_approve_commands, got nil")
	}
	if err := cfg.Set("log_level", "verbose"); err == nil {
		t.Error("Expected error for invalid log_level, got nil")
	}
}

func TestGetConfigDir(t *testing.T) {
	dir := GetConfigDir()
	if dir == "" {
		t.Error("Expected non-empty config directory")
	}
}

func TestGetConfigFile(t *testing.T) {
	file := GetConfigFile()
	if file == "" {
		t.Error("Expected non-empty config file path")
	}
	if filepath.Ext(file) != ".json" {
		t.Errorf("Expected config file to have .json extension, got '%s'", filepath.Ext(file))
	}
}
