package curator

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config holds all configuration for the curator application
type Config struct {
	AI         AIConfig         `json:"ai"`
	FileSystem FileSystemConfig `json:"filesystem"`
}

// AIConfig holds AI-related configuration
type AIConfig struct {
	Provider string         `json:"provider"` // "mock" or "gemini"
	Gemini   *GeminiConfig  `json:"gemini,omitempty"`
}

// FileSystemConfig holds filesystem-related configuration
type FileSystemConfig struct {
	Type        string                `json:"type"`        // "memory", "local", or "googledrive"
	Root        string                `json:"root"`        // Root path for local filesystem
	GoogleDrive *GoogleDriveConfig    `json:"googledrive,omitempty"`
}

// LoadConfig loads configuration from environment variables and defaults
func LoadConfig() *Config {
	config := &Config{
		AI: AIConfig{
			Provider: getEnvOrDefault("CURATOR_AI_PROVIDER", "mock"),
		},
		FileSystem: FileSystemConfig{
			Type: getEnvOrDefault("CURATOR_FILESYSTEM_TYPE", "local"),
			Root: getEnvOrDefault("CURATOR_FILESYSTEM_ROOT", "."),
		},
	}
	
	// Load Gemini config if provider is gemini
	if config.AI.Provider == "gemini" {
		config.AI.Gemini = loadGeminiConfig()
	}
	
	// Load Google Drive config if filesystem is googledrive
	if config.FileSystem.Type == "googledrive" {
		config.FileSystem.GoogleDrive = loadGoogleDriveConfig()
	}
	
	return config
}

// loadGeminiConfig loads Gemini configuration from environment
func loadGeminiConfig() *GeminiConfig {
	config := DefaultGeminiConfig()
	
	// Load API key from environment
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}
	
	// Load model from environment
	if model := os.Getenv("GEMINI_MODEL"); model != "" {
		config.Model = model
	}
	
	// Load max tokens from environment
	if maxTokensStr := os.Getenv("GEMINI_MAX_TOKENS"); maxTokensStr != "" {
		if maxTokens, err := strconv.ParseInt(maxTokensStr, 10, 32); err == nil {
			config.MaxTokens = int32(maxTokens)
		} else {
			log.Printf("Warning: invalid GEMINI_MAX_TOKENS value '%s', using default: %v", maxTokensStr, err)
		}
	}
	
	// Load timeout from environment
	if timeoutStr := os.Getenv("GEMINI_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.Timeout = timeout
		} else {
			log.Printf("Warning: invalid GEMINI_TIMEOUT value '%s', using default: %v", timeoutStr, err)
		}
	}
	
	return config
}

// loadGoogleDriveConfig loads Google Drive configuration from environment
func loadGoogleDriveConfig() *GoogleDriveConfig {
	config := DefaultGoogleDriveConfig()
	
	// Load OAuth2 credentials file path from environment
	if credFile := os.Getenv("GOOGLE_DRIVE_OAUTH_CREDENTIALS"); credFile != "" {
		config.OAuth2CredentialsFile = credFile
	}
	
	// Load OAuth2 token file path from environment (optional)
	if tokenFile := os.Getenv("GOOGLE_DRIVE_OAUTH_TOKENS"); tokenFile != "" {
		config.OAuth2TokenFile = tokenFile
	}
	
	// Load root folder ID from environment
	if rootID := os.Getenv("GOOGLE_DRIVE_ROOT_FOLDER_ID"); rootID != "" {
		config.RootFolderID = rootID
	}
	
	// Load application name from environment
	if appName := os.Getenv("GOOGLE_DRIVE_APPLICATION_NAME"); appName != "" {
		config.ApplicationName = appName
	}
	
	return config
}

// CreateAnalyzer creates an AI analyzer based on configuration
func (c *Config) CreateAnalyzer() (AIAnalyzer, error) {
	switch c.AI.Provider {
	case "mock":
		return NewMockAIAnalyzer(), nil
	case "gemini":
		if c.AI.Gemini == nil {
			return nil, fmt.Errorf("Gemini configuration is required when provider is 'gemini'")
		}
		return NewGeminiAnalyzer(c.AI.Gemini)
	default:
		return nil, fmt.Errorf("unknown AI provider: %s", c.AI.Provider)
	}
}

// CreateFileSystem creates a filesystem based on configuration
func (c *Config) CreateFileSystem() (FileSystem, error) {
	switch c.FileSystem.Type {
	case "memory":
		return NewMemoryFileSystem(), nil
	case "local":
		return NewLocalFileSystem(c.FileSystem.Root)
	case "googledrive":
		if c.FileSystem.GoogleDrive == nil {
			return nil, fmt.Errorf("Google Drive configuration is required when filesystem is 'googledrive'")
		}
		return NewGoogleDriveFileSystem(c.FileSystem.GoogleDrive)
	default:
		return nil, fmt.Errorf("unknown filesystem type: %s", c.FileSystem.Type)
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate AI config
	switch c.AI.Provider {
	case "mock":
		// Mock analyzer needs no validation
	case "gemini":
		if c.AI.Gemini == nil {
			return fmt.Errorf("Gemini configuration is required when provider is 'gemini'")
		}
		if c.AI.Gemini.APIKey == "" {
			return fmt.Errorf("Gemini API key is required (set GEMINI_API_KEY environment variable)")
		}
	default:
		return fmt.Errorf("unknown AI provider: %s (valid options: mock, gemini)", c.AI.Provider)
	}
	
	// Validate filesystem config
	switch c.FileSystem.Type {
	case "memory":
		// Memory filesystem needs no validation
	case "local":
		if c.FileSystem.Root == "" {
			return fmt.Errorf("local filesystem root path is required")
		}
	case "googledrive":
		if c.FileSystem.GoogleDrive == nil {
			return fmt.Errorf("Google Drive configuration is required when filesystem is 'googledrive'")
		}
		if c.FileSystem.GoogleDrive.OAuth2CredentialsFile == "" {
			return fmt.Errorf("Google Drive OAuth2 credentials file is required (set GOOGLE_DRIVE_OAUTH_CREDENTIALS environment variable)")
		}
	default:
		return fmt.Errorf("unknown filesystem type: %s (valid options: memory, local, googledrive)", c.FileSystem.Type)
	}
	
	return nil
}

// GetDefaultStoreDir returns the default directory for storing plans and operation logs
func GetDefaultStoreDir() string {
	// Check environment variable first
	if storeDir := os.Getenv("CURATOR_STORE_DIR"); storeDir != "" {
		return storeDir
	}
	
	// Use user's home directory by default
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home dir is not available
		return ".curator"
	}
	
	return filepath.Join(homeDir, ".curator")
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}