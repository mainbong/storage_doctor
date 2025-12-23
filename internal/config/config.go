package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mainbong/storage_doctor/internal/filesystem"
)

// Config holds the application configuration
type Config struct {
	LLMProvider string `json:"llm_provider"` // "anthropic" or "openai"
	Anthropic   struct {
		APIKey string `json:"api_key"`
		Model  string `json:"model"` // e.g., "claude-3-5-sonnet-20241022"
	} `json:"anthropic"`
	OpenAI struct {
		APIKey string `json:"api_key"`
		Model  string `json:"model"` // e.g., "gpt-4-turbo-preview"
	} `json:"openai"`
	Search struct {
		Provider string `json:"provider"` // "google", "bing", "duckduckgo", "serper"
		Google   struct {
			APIKey string `json:"api_key"`
			CX     string `json:"cx"` // Custom Search Engine ID
		} `json:"google"`
		Bing struct {
			APIKey string `json:"api_key"`
		} `json:"bing"`
		Serper struct {
			APIKey string `json:"api_key"`
		} `json:"serper"`
	} `json:"search"`
	AutoApproveCommands bool   `json:"auto_approve_commands"` // Auto-approve all commands
	SessionDir          string `json:"session_dir"`
	BackupDir           string `json:"backup_dir"`
	LogDir              string `json:"log_dir"`
	LogLevel            string `json:"log_level"` // "debug", "info", "warn", "error"
}

var (
	configDir  = filepath.Join(os.Getenv("HOME"), ".storage-doctor")
	configFile = filepath.Join(configDir, "config.json")
	defaultFS  = filesystem.NewOSFileSystem()
)

// Load loads the configuration from file or creates a default one
func Load() (*Config, error) {
	return LoadWithFS(defaultFS, configDir, configFile)
}

// LoadWithFS loads the configuration using a custom FileSystem (for testing)
func LoadWithFS(fs filesystem.FileSystem, dir, file string) (*Config, error) {
	// Create config directory if it doesn't exist
	if err := fs.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	cfg := &Config{}

	// Set defaults
	cfg.LLMProvider = "anthropic"
	cfg.Anthropic.Model = "claude-haiku-4-5-20251001"
	cfg.OpenAI.Model = "gpt-5"
	cfg.Search.Provider = "duckduckgo"
	cfg.AutoApproveCommands = false
	cfg.SessionDir = filepath.Join(dir, "sessions")
	cfg.BackupDir = filepath.Join(dir, "backups")
	cfg.LogDir = filepath.Join(dir, "logs")
	cfg.LogLevel = "info"

	// Try to load from file
	if _, err := fs.Stat(file); err == nil {
		data, err := fs.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	} else {
		// Config file doesn't exist, prefer env keys before saving defaults
		if err := cfg.loadAPIKeysFromEnv(fs, file); err != nil {
			return nil, err
		}
		if err := cfg.SaveWithFS(fs, file); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
	}

	// Load API keys with priority: config.json -> environment variables -> prompt user
	if err := cfg.loadAPIKeysFromEnv(fs, file); err != nil {
		return nil, err
	}

	// Ensure session, backup, and log directories exist
	if err := cfg.ensureDir(fs, file, "session_dir", &cfg.SessionDir, filepath.Join(dir, "sessions")); err != nil {
		return nil, err
	}
	if err := cfg.ensureDir(fs, file, "backup_dir", &cfg.BackupDir, filepath.Join(dir, "backups")); err != nil {
		return nil, err
	}
	if err := cfg.ensureDir(fs, file, "log_dir", &cfg.LogDir, filepath.Join(dir, "logs")); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save saves the configuration to file
func (c *Config) Save() error {
	return c.SaveWithFS(defaultFS, configFile)
}

// SaveWithFS saves the configuration using a custom FileSystem (for testing)
func (c *Config) SaveWithFS(fs filesystem.FileSystem, file string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(file)
	if err := fs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := fs.WriteFile(file, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() string {
	return configDir
}

// GetConfigFile returns the configuration file path
func GetConfigFile() string {
	return configFile
}

// loadAPIKeys loads API keys with priority: config.json -> environment variables -> prompt user
func (c *Config) loadAPIKeysFromEnv(fs filesystem.FileSystem, file string) error {
	// Load Anthropic API key
	if c.Anthropic.APIKey == "" {
		// Try environment variable
		if envKey := os.Getenv("ANTHROPIC_API_KEY"); envKey != "" {
			c.Anthropic.APIKey = envKey
			// Save to config file if loaded from env
			if err := c.SaveWithFS(fs, file); err != nil {
				return fmt.Errorf("failed to save anthropic api key: %w", err)
			}
		}
		// Note: User prompt is handled in main() via ensureAPIKeys()
	}

	// Load OpenAI API key
	if c.OpenAI.APIKey == "" {
		// Try environment variable
		if envKey := os.Getenv("OPENAI_API_KEY"); envKey != "" {
			c.OpenAI.APIKey = envKey
			// Save to config file if loaded from env
			if err := c.SaveWithFS(fs, file); err != nil {
				return fmt.Errorf("failed to save openai api key: %w", err)
			}
		}
		// Note: User prompt is handled in main() via ensureAPIKeys()
	}

	return nil
}

// promptAPIKey prompts user for API key input
func promptAPIKey(keyName, envVarName string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\n%s가 설정되지 않았습니다.\n", keyName)
	fmt.Printf("환경변수 %s를 설정하거나 아래에 입력해주세요.\n", envVarName)
	fmt.Printf("%s (엔터만 누르면 건너뜀): ", keyName)

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input != "" {
		fmt.Printf("%s가 설정되었습니다.\n", keyName)
	}

	return input
}

func (c *Config) ensureDir(fs filesystem.FileSystem, file, key string, value *string, fallback string) error {
	if strings.TrimSpace(*value) == "" {
		*value = fallback
		if err := c.SaveWithFS(fs, file); err != nil {
			return fmt.Errorf("failed to save default %s: %w", key, err)
		}
	}

	if err := fs.MkdirAll(*value, 0755); err != nil {
		*value = fallback
		if err := fs.MkdirAll(*value, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", key, err)
		}
		if err := c.SaveWithFS(fs, file); err != nil {
			return fmt.Errorf("failed to save fallback %s: %w", key, err)
		}
	}

	return nil
}

// Set updates a config value by key.
func (c *Config) Set(key, value string) error {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "llm_provider":
		if value != "anthropic" && value != "openai" {
			return fmt.Errorf("invalid llm_provider: %s", value)
		}
		c.LLMProvider = value
	case "anthropic.api_key":
		c.Anthropic.APIKey = value
	case "anthropic.model":
		c.Anthropic.Model = value
	case "openai.api_key":
		c.OpenAI.APIKey = value
	case "openai.model":
		c.OpenAI.Model = value
	case "search.provider":
		if value != "google" && value != "bing" && value != "duckduckgo" && value != "serper" {
			return fmt.Errorf("invalid search.provider: %s", value)
		}
		c.Search.Provider = value
	case "search.google.api_key":
		c.Search.Google.APIKey = value
	case "search.google.cx":
		c.Search.Google.CX = value
	case "search.bing.api_key":
		c.Search.Bing.APIKey = value
	case "search.serper.api_key":
		c.Search.Serper.APIKey = value
	case "auto_approve_commands":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid auto_approve_commands: %s", value)
		}
		c.AutoApproveCommands = parsed
	case "session_dir":
		c.SessionDir = value
	case "backup_dir":
		c.BackupDir = value
	case "log_dir":
		c.LogDir = value
	case "log_level":
		switch strings.ToLower(value) {
		case "debug", "info", "warn", "error":
			c.LogLevel = strings.ToLower(value)
		default:
			return fmt.Errorf("invalid log_level: %s", value)
		}
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return nil
}
