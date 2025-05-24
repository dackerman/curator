package curator

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestGeminiConfig_Defaults(t *testing.T) {
	config := DefaultGeminiConfig()
	
	if config.Model != "gemini-1.5-flash" {
		t.Errorf("Expected default model 'gemini-1.5-flash', got %s", config.Model)
	}
	
	if config.MaxTokens != 8192 {
		t.Errorf("Expected default max tokens 8192, got %d", config.MaxTokens)
	}
	
	if config.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", config.Timeout)
	}
	
	if config.RateLimit != 1.0 {
		t.Errorf("Expected default rate limit 1.0, got %f", config.RateLimit)
	}
	
	if config.MaxRetries != 3 {
		t.Errorf("Expected default max retries 3, got %d", config.MaxRetries)
	}
}

func TestNewGeminiAnalyzer_RequiresAPIKey(t *testing.T) {
	config := DefaultGeminiConfig()
	config.APIKey = "" // Empty API key
	
	_, err := NewGeminiAnalyzer(config)
	if err == nil {
		t.Error("Expected error when API key is empty")
	}
	
	expectedMsg := "Gemini API key is required"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestConfig_LoadConfig(t *testing.T) {
	// Save original env vars
	originalProvider := os.Getenv("CURATOR_AI_PROVIDER")
	originalAPIKey := os.Getenv("GEMINI_API_KEY")
	
	// Cleanup
	defer func() {
		if originalProvider != "" {
			os.Setenv("CURATOR_AI_PROVIDER", originalProvider)
		} else {
			os.Unsetenv("CURATOR_AI_PROVIDER")
		}
		if originalAPIKey != "" {
			os.Setenv("GEMINI_API_KEY", originalAPIKey)
		} else {
			os.Unsetenv("GEMINI_API_KEY")
		}
	}()
	
	// Test default (mock) provider
	os.Unsetenv("CURATOR_AI_PROVIDER")
	config := LoadConfig()
	
	if config.AI.Provider != "mock" {
		t.Errorf("Expected default provider 'mock', got %s", config.AI.Provider)
	}
	
	// Test gemini provider
	os.Setenv("CURATOR_AI_PROVIDER", "gemini")
	os.Setenv("GEMINI_API_KEY", "test-key")
	config = LoadConfig()
	
	if config.AI.Provider != "gemini" {
		t.Errorf("Expected provider 'gemini', got %s", config.AI.Provider)
	}
	
	if config.AI.Gemini.APIKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got %s", config.AI.Gemini.APIKey)
	}
}

func TestConfig_Validate(t *testing.T) {
	// Test valid mock config
	config := &Config{
		AI: AIConfig{
			Provider: "mock",
		},
		FileSystem: FileSystemConfig{
			Type: "memory",
		},
	}
	
	if err := config.Validate(); err != nil {
		t.Errorf("Mock config should be valid: %v", err)
	}
	
	// Test valid gemini config
	config = &Config{
		AI: AIConfig{
			Provider: "gemini",
			Gemini: &GeminiConfig{
				APIKey: "test-key",
			},
		},
		FileSystem: FileSystemConfig{
			Type: "memory",
		},
	}
	
	if err := config.Validate(); err != nil {
		t.Errorf("Gemini config with API key should be valid: %v", err)
	}
	
	// Test invalid gemini config (no API key)
	config = &Config{
		AI: AIConfig{
			Provider: "gemini",
			Gemini: &GeminiConfig{
				APIKey: "",
			},
		},
		FileSystem: FileSystemConfig{
			Type: "memory",
		},
	}
	
	if err := config.Validate(); err == nil {
		t.Error("Gemini config without API key should be invalid")
	}
	
	// Test invalid provider
	config = &Config{
		AI: AIConfig{
			Provider: "unknown",
		},
		FileSystem: FileSystemConfig{
			Type: "memory",
		},
	}
	
	if err := config.Validate(); err == nil {
		t.Error("Unknown provider should be invalid")
	}
}

func TestConfig_CreateAnalyzer(t *testing.T) {
	// Test creating mock analyzer
	config := &Config{
		AI: AIConfig{
			Provider: "mock",
		},
	}
	
	analyzer, err := config.CreateAnalyzer()
	if err != nil {
		t.Fatalf("Failed to create mock analyzer: %v", err)
	}
	
	if _, ok := analyzer.(*MockAIAnalyzer); !ok {
		t.Error("Expected MockAIAnalyzer")
	}
	
	// Test creating gemini analyzer (should succeed with fake API key, actual API calls will fail later)
	config = &Config{
		AI: AIConfig{
			Provider: "gemini",
			Gemini: &GeminiConfig{
				APIKey: "fake-key-for-testing",
			},
		},
	}
	
	analyzer, err = config.CreateAnalyzer()
	if err != nil {
		t.Errorf("Should be able to create Gemini analyzer with fake API key: %v", err)
	}
	
	if geminiAnalyzer, ok := analyzer.(*GeminiAnalyzer); ok {
		geminiAnalyzer.Close() // Clean up
	} else {
		t.Error("Expected GeminiAnalyzer")
	}
}

// Integration test - only runs if GEMINI_API_KEY is set
func TestGeminiAnalyzer_Integration(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test - GEMINI_API_KEY not set")
	}
	
	config := DefaultGeminiConfig()
	config.APIKey = apiKey
	config.RateLimit = 0.5 // Be extra conservative in tests
	
	analyzer, err := NewGeminiAnalyzer(config)
	if err != nil {
		t.Fatalf("Failed to create Gemini analyzer: %v", err)
	}
	defer analyzer.Close()
	
	// Test with a simple file structure
	fs := NewMemoryFileSystem()
	fs.AddFile("/document.pdf", []byte("test document"), "application/pdf")
	fs.AddFile("/image.jpg", []byte("test image"), "image/jpeg")
	
	files, err := fs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
	// Test reorganization
	plan, err := analyzer.AnalyzeForReorganization(files)
	if err != nil {
		t.Fatalf("Failed to analyze for reorganization: %v", err)
	}
	
	if plan.ID == "" {
		t.Error("Plan should have an ID")
	}
	
	if len(plan.Moves) == 0 {
		t.Error("Plan should have moves")
	}
	
	// Basic validation that the response was parsed correctly
	if plan.Summary.FoldersCreated < 0 {
		t.Error("Folders created should not be negative")
	}
	
	if plan.Rationale == "" {
		t.Error("Plan should have a rationale")
	}
	
	t.Logf("Integration test passed - created plan with %d moves", len(plan.Moves))
}

func TestGeminiAnalyzer_PromptBuilding(t *testing.T) {
	config := DefaultGeminiConfig()
	config.APIKey = "fake-key" // We won't actually call the API
	
	analyzer, err := NewGeminiAnalyzer(config)
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}
	defer analyzer.Close()
	
	// Test file data
	fs := NewMemoryFileSystem()
	fs.AddFile("/document.pdf", []byte("test"), "application/pdf")
	fs.AddFile("/image.jpg", []byte("test"), "image/jpeg")
	
	files, _ := fs.List("/")
	
	// Test reorganization prompt
	prompt := analyzer.buildReorganizationPrompt(files)
	if prompt == "" {
		t.Error("Reorganization prompt should not be empty")
	}
	
	if !containsSubstring(prompt, "FILE: /document.pdf") {
		t.Error("Prompt should contain file information")
	}
	
	if !containsSubstring(prompt, "JSON object") {
		t.Error("Prompt should request JSON response")
	}
	
	// Test duplication prompt
	prompt = analyzer.buildDuplicationPrompt(files)
	if prompt == "" {
		t.Error("Duplication prompt should not be empty")
	}
	
	if !containsSubstring(prompt, "duplicates") {
		t.Error("Duplication prompt should mention duplicates")
	}
	
	// Test cleanup prompt
	prompt = analyzer.buildCleanupPrompt(files)
	if prompt == "" {
		t.Error("Cleanup prompt should not be empty")
	}
	
	if !containsSubstring(prompt, "junk files") {
		t.Error("Cleanup prompt should mention junk files")
	}
	
	// Test renaming prompt
	prompt = analyzer.buildRenamingPrompt(files)
	if prompt == "" {
		t.Error("Renaming prompt should not be empty")
	}
	
	if !containsSubstring(prompt, "consistency") {
		t.Error("Renaming prompt should mention consistency")
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(str, substr string) bool {
	return len(str) >= len(substr) && 
		   len(substr) > 0 && 
		   strings.Contains(str, substr)
}