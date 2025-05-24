package curator

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the curator application
type Config struct {
	AI AIConfig `json:"ai"`
}

// AIConfig holds AI-related configuration
type AIConfig struct {
	Provider string         `json:"provider"` // "mock" or "gemini"
	Gemini   *GeminiConfig  `json:"gemini,omitempty"`
}

// LoadConfig loads configuration from environment variables and defaults
func LoadConfig() *Config {
	config := &Config{
		AI: AIConfig{
			Provider: getEnvOrDefault("CURATOR_AI_PROVIDER", "mock"),
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
		}
	}
	
	// Load timeout from environment
	if timeoutStr := os.Getenv("GEMINI_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.Timeout = timeout
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

// Validate validates the configuration
func (c *Config) Validate() error {
	switch c.AI.Provider {
	case "mock":
		// Mock analyzer needs no validation
		return nil
	case "gemini":
		if c.AI.Gemini == nil {
			return fmt.Errorf("Gemini configuration is required when provider is 'gemini'")
		}
		if c.AI.Gemini.APIKey == "" {
			return fmt.Errorf("Gemini API key is required (set GEMINI_API_KEY environment variable)")
		}
		return nil
	default:
		return fmt.Errorf("unknown AI provider: %s (valid options: mock, gemini)", c.AI.Provider)
	}
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}