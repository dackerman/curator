package curator

import (
	"fmt"
	"log"
	"os"
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
	Type string `json:"type"` // "memory" or "local"
	Root string `json:"root"` // Root path for local filesystem
}

// LoadConfig loads configuration from environment variables and defaults
func LoadConfig() *Config {
	config := &Config{
		AI: AIConfig{
			Provider: getEnvOrDefault("CURATOR_AI_PROVIDER", "mock"),
		},
		FileSystem: FileSystemConfig{
			Type: getEnvOrDefault("CURATOR_FILESYSTEM_TYPE", "memory"),
			Root: getEnvOrDefault("CURATOR_FILESYSTEM_ROOT", "."),
		},
	}
	
	// Load Gemini config if provider is gemini
	if config.AI.Provider == "gemini" {
		config.AI.Gemini = loadGeminiConfig()
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
	default:
		return fmt.Errorf("unknown filesystem type: %s (valid options: memory, local)", c.FileSystem.Type)
	}
	
	return nil
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}