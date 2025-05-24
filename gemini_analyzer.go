package curator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"golang.org/x/time/rate"
)

// GeminiConfig holds configuration for Gemini AI analyzer
type GeminiConfig struct {
	APIKey       string
	Model        string
	MaxTokens    int32
	Timeout      time.Duration
	RateLimit    float64 // requests per second
	MaxRetries   int
	RetryDelay   time.Duration
}

// DefaultGeminiConfig returns default configuration for Gemini
func DefaultGeminiConfig() *GeminiConfig {
	return &GeminiConfig{
		Model:      "gemini-1.5-flash",
		MaxTokens:  8192,
		Timeout:    30 * time.Second,
		RateLimit:  1.0, // 1 request per second to be conservative
		MaxRetries: 3,
		RetryDelay: 2 * time.Second,
	}
}

// GeminiAnalyzer implements AIAnalyzer interface using Gemini AI
type GeminiAnalyzer struct {
	config  *GeminiConfig
	client  *genai.Client
	limiter *rate.Limiter
}

// NewGeminiAnalyzer creates a new Gemini AI analyzer
func NewGeminiAnalyzer(config *GeminiConfig) (*GeminiAnalyzer, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}
	
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(config.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	
	// Create rate limiter
	limiter := rate.NewLimiter(rate.Limit(config.RateLimit), 1)
	
	return &GeminiAnalyzer{
		config:  config,
		client:  client,
		limiter: limiter,
	}, nil
}

// Close closes the Gemini client connection
func (g *GeminiAnalyzer) Close() error {
	return g.client.Close()
}

// AnalyzeForReorganization implements AIAnalyzer.AnalyzeForReorganization
func (g *GeminiAnalyzer) AnalyzeForReorganization(files []FileInfo) (*ReorganizationPlan, error) {
	prompt := g.buildReorganizationPrompt(files)
	
	response, err := g.callGemini(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini for reorganization: %w", err)
	}
	
	plan, err := g.parseReorganizationResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reorganization response: %w", err)
	}
	
	return plan, nil
}

// AnalyzeForDuplicates implements AIAnalyzer.AnalyzeForDuplicates
func (g *GeminiAnalyzer) AnalyzeForDuplicates(files []FileInfo) (*DuplicationReport, error) {
	prompt := g.buildDuplicationPrompt(files)
	
	response, err := g.callGemini(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini for duplicates: %w", err)
	}
	
	report, err := g.parseDuplicationResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duplication response: %w", err)
	}
	
	return report, nil
}

// AnalyzeForCleanup implements AIAnalyzer.AnalyzeForCleanup
func (g *GeminiAnalyzer) AnalyzeForCleanup(files []FileInfo) (*CleanupPlan, error) {
	prompt := g.buildCleanupPrompt(files)
	
	response, err := g.callGemini(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini for cleanup: %w", err)
	}
	
	plan, err := g.parseCleanupResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cleanup response: %w", err)
	}
	
	return plan, nil
}

// AnalyzeForRenaming implements AIAnalyzer.AnalyzeForRenaming
func (g *GeminiAnalyzer) AnalyzeForRenaming(files []FileInfo) (*RenamingPlan, error) {
	prompt := g.buildRenamingPrompt(files)
	
	response, err := g.callGemini(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini for renaming: %w", err)
	}
	
	plan, err := g.parseRenamingResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse renaming response: %w", err)
	}
	
	return plan, nil
}

// callGemini makes a request to Gemini API with rate limiting and retries
func (g *GeminiAnalyzer) callGemini(prompt string) (string, error) {
	var lastErr error
	
	for attempt := 0; attempt < g.config.MaxRetries; attempt++ {
		// Wait for rate limiter
		ctx := context.Background()
		if err := g.limiter.Wait(ctx); err != nil {
			return "", fmt.Errorf("rate limiter error: %w", err)
		}
		
		response, err := g.makeRequest(prompt)
		if err == nil {
			return response, nil
		}
		
		lastErr = err
		
		// If this is the last attempt, don't wait
		if attempt < g.config.MaxRetries-1 {
			time.Sleep(g.config.RetryDelay)
		}
	}
	
	return "", fmt.Errorf("failed after %d attempts: %w", g.config.MaxRetries, lastErr)
}

// makeRequest makes a single request to Gemini API
func (g *GeminiAnalyzer) makeRequest(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.config.Timeout)
	defer cancel()
	
	model := g.client.GenerativeModel(g.config.Model)
	model.SetMaxOutputTokens(g.config.MaxTokens)
	model.SetTemperature(0.7) // Balanced creativity vs consistency
	
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}
	
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no response candidates from Gemini")
	}
	
	if resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("empty content in Gemini response")
	}
	
	var response strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			response.WriteString(string(text))
		}
	}
	
	return response.String(), nil
}

// buildReorganizationPrompt creates a prompt for file reorganization
func (g *GeminiAnalyzer) buildReorganizationPrompt(files []FileInfo) string {
	var filesInfo strings.Builder
	filesInfo.WriteString("Files to analyze:\n")
	
	for _, file := range files {
		if file.IsDir() {
			filesInfo.WriteString(fmt.Sprintf("FOLDER: %s\n", file.Path()))
		} else {
			filesInfo.WriteString(fmt.Sprintf("FILE: %s (size: %d bytes, type: %s)\n", 
				file.Path(), file.Size(), file.MimeType()))
		}
	}
	
	return fmt.Sprintf(`You are an expert file organization assistant. Analyze the following file structure and create an intelligent reorganization plan.

%s

Create a reorganization plan that:
1. Groups related files together logically
2. Creates a clear folder hierarchy 
3. Reduces clutter in the root directory
4. Makes files easier to find
5. Follows common organizational patterns (Documents, Images, Videos, etc.)

Respond with a JSON object in exactly this format:
{
  "id": "reorg-<timestamp>",
  "moves": [
    {
      "id": "move-1",
      "source": "/path/to/source",
      "destination": "/path/to/destination", 
      "reason": "Clear explanation of why this move makes sense",
      "type": "CREATE_FOLDER|FILE_MOVE|FOLDER_MOVE",
      "fileCount": 1
    }
  ],
  "summary": {
    "foldersCreated": 5,
    "filesMoved": 20,
    "foldersMovedDeduplicated": 2,
    "depthReduction": "25%%",
    "organizationImprovement": "85%% of files will be in semantically organized folders"
  },
  "rationale": "Overall explanation of the reorganization strategy"
}

Important:
- CREATE_FOLDER moves should come before moves that use those folders
- Provide clear, helpful reasons for each move
- Focus on practical, logical organization
- Avoid moving files that are already well-organized`, filesInfo.String())
}

// buildDuplicationPrompt creates a prompt for duplicate detection
func (g *GeminiAnalyzer) buildDuplicationPrompt(files []FileInfo) string {
	var filesInfo strings.Builder
	filesInfo.WriteString("Files to analyze for duplicates:\n")
	
	for _, file := range files {
		if !file.IsDir() {
			filesInfo.WriteString(fmt.Sprintf("FILE: %s (size: %d bytes, hash: %s)\n", 
				file.Path(), file.Size(), file.Hash()))
		}
	}
	
	return fmt.Sprintf(`You are analyzing files for duplicates. Files with the same hash are identical.

%s

Identify duplicate files and respond with a JSON object in exactly this format:
{
  "id": "dup-<timestamp>",
  "duplicates": [
    {
      "hash": "abc123",
      "files": ["/path/to/file1", "/path/to/file2"],
      "size": 1024
    }
  ],
  "summary": {
    "totalDuplicates": 3,
    "spaceSaved": 3072
  }
}

Only include groups where there are 2+ files with the same hash.`, filesInfo.String())
}

// buildCleanupPrompt creates a prompt for cleanup analysis
func (g *GeminiAnalyzer) buildCleanupPrompt(files []FileInfo) string {
	var filesInfo strings.Builder
	filesInfo.WriteString("Files to analyze for cleanup:\n")
	
	for _, file := range files {
		if !file.IsDir() {
			filesInfo.WriteString(fmt.Sprintf("FILE: %s (size: %d bytes, type: %s)\n", 
				file.Path(), file.Size(), file.MimeType()))
		}
	}
	
	return fmt.Sprintf(`You are analyzing files to identify those that can be safely deleted (junk files).

%s

Identify files that are likely safe to delete, such as:
- Temporary files (.tmp, .temp, .cache)
- Empty files (0 bytes)
- Backup files (.bak, .backup, ~)
- System junk files
- Log files that are very old
- Cache files

Be conservative - only suggest files that are very likely to be safe to delete.

Respond with a JSON object in exactly this format:
{
  "id": "cleanup-<timestamp>",
  "deletions": [
    {
      "id": "del-1",
      "path": "/path/to/file",
      "reason": "Why this file is safe to delete",
      "size": 1024
    }
  ],
  "summary": {
    "filesDeleted": 5,
    "spaceFreed": 5120
  }
}`, filesInfo.String())
}

// buildRenamingPrompt creates a prompt for file renaming
func (g *GeminiAnalyzer) buildRenamingPrompt(files []FileInfo) string {
	var filesInfo strings.Builder
	filesInfo.WriteString("Files to analyze for renaming:\n")
	
	for _, file := range files {
		if !file.IsDir() {
			filesInfo.WriteString(fmt.Sprintf("FILE: %s\n", file.Name()))
		}
	}
	
	return fmt.Sprintf(`You are analyzing filenames to standardize them for consistency.

%s

Suggest renames to improve filename consistency by:
- Removing or replacing spaces with underscores/hyphens
- Standardizing case (preferably lowercase)
- Removing special characters
- Making names more descriptive where obvious

Only suggest renames that genuinely improve the filename quality.

Respond with a JSON object in exactly this format:
{
  "id": "rename-<timestamp>",
  "renames": [
    {
      "id": "rename-1",
      "oldName": "My Document.pdf",
      "newName": "my_document.pdf",
      "reason": "Standardize to lowercase with underscores"
    }
  ],
  "summary": {
    "filesRenamed": 3,
    "pattern": "lowercase_with_underscores"
  }
}`, filesInfo.String())
}

// parseReorganizationResponse parses Gemini's response into a ReorganizationPlan
func (g *GeminiAnalyzer) parseReorganizationResponse(response string) (*ReorganizationPlan, error) {
	// Extract JSON from response (in case there's extra text)
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}") + 1
	
	if jsonStart == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no valid JSON found in response")
	}
	
	jsonStr := response[jsonStart:jsonEnd]
	
	var result struct {
		ID        string `json:"id"`
		Moves     []struct {
			ID          string `json:"id"`
			Source      string `json:"source"`
			Destination string `json:"destination"`
			Reason      string `json:"reason"`
			Type        string `json:"type"`
			FileCount   int    `json:"fileCount"`
		} `json:"moves"`
		Summary struct {
			FoldersCreated              int    `json:"foldersCreated"`
			FilesMoved                  int    `json:"filesMoved"`
			FoldersMovedDeduplicated    int    `json:"foldersMovedDeduplicated"`
			DepthReduction              string `json:"depthReduction"`
			OrganizationImprovement     string `json:"organizationImprovement"`
		} `json:"summary"`
		Rationale string `json:"rationale"`
	}
	
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	
	// Convert to our types
	moves := make([]Move, len(result.Moves))
	for i, m := range result.Moves {
		var moveType MoveType
		switch m.Type {
		case "CREATE_FOLDER":
			moveType = CreateFolder
		case "FILE_MOVE":
			moveType = FileMove
		case "FOLDER_MOVE":
			moveType = FolderMove
		default:
			return nil, fmt.Errorf("unknown move type: %s", m.Type)
		}
		
		moves[i] = Move{
			ID:          m.ID,
			Source:      m.Source,
			Destination: m.Destination,
			Reason:      m.Reason,
			Type:        moveType,
			FileCount:   m.FileCount,
		}
	}
	
	plan := &ReorganizationPlan{
		ID:        result.ID,
		Timestamp: time.Now(),
		Moves:     moves,
		Summary: Summary{
			FoldersCreated:              result.Summary.FoldersCreated,
			FilesMoved:                  result.Summary.FilesMoved,
			FoldersMovedDeduplicated:    result.Summary.FoldersMovedDeduplicated,
			DepthReduction:              result.Summary.DepthReduction,
			OrganizationImprovement:     result.Summary.OrganizationImprovement,
		},
		Rationale: result.Rationale,
	}
	
	return plan, nil
}

// parseDuplicationResponse parses Gemini's response into a DuplicationReport
func (g *GeminiAnalyzer) parseDuplicationResponse(response string) (*DuplicationReport, error) {
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}") + 1
	
	if jsonStart == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no valid JSON found in response")
	}
	
	jsonStr := response[jsonStart:jsonEnd]
	
	var result struct {
		ID         string `json:"id"`
		Duplicates []struct {
			Hash  string   `json:"hash"`
			Files []string `json:"files"`
			Size  int64    `json:"size"`
		} `json:"duplicates"`
		Summary struct {
			TotalDuplicates int   `json:"totalDuplicates"`
			SpaceSaved      int64 `json:"spaceSaved"`
		} `json:"summary"`
	}
	
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	
	duplicates := make([]DuplicateGroup, len(result.Duplicates))
	for i, d := range result.Duplicates {
		duplicates[i] = DuplicateGroup{
			Hash:  d.Hash,
			Files: d.Files,
			Size:  d.Size,
		}
	}
	
	report := &DuplicationReport{
		ID:         result.ID,
		Timestamp:  time.Now(),
		Duplicates: duplicates,
		Summary: DuplicationSummary{
			TotalDuplicates: result.Summary.TotalDuplicates,
			SpaceSaved:      result.Summary.SpaceSaved,
		},
	}
	
	return report, nil
}

// parseCleanupResponse parses Gemini's response into a CleanupPlan
func (g *GeminiAnalyzer) parseCleanupResponse(response string) (*CleanupPlan, error) {
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}") + 1
	
	if jsonStart == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no valid JSON found in response")
	}
	
	jsonStr := response[jsonStart:jsonEnd]
	
	var result struct {
		ID        string `json:"id"`
		Deletions []struct {
			ID     string `json:"id"`
			Path   string `json:"path"`
			Reason string `json:"reason"`
			Size   int64  `json:"size"`
		} `json:"deletions"`
		Summary struct {
			FilesDeleted int   `json:"filesDeleted"`
			SpaceFreed   int64 `json:"spaceFreed"`
		} `json:"summary"`
	}
	
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	
	deletions := make([]Deletion, len(result.Deletions))
	for i, d := range result.Deletions {
		deletions[i] = Deletion{
			ID:     d.ID,
			Path:   d.Path,
			Reason: d.Reason,
			Size:   d.Size,
		}
	}
	
	plan := &CleanupPlan{
		ID:        result.ID,
		Timestamp: time.Now(),
		Deletions: deletions,
		Summary: CleanupSummary{
			FilesDeleted: result.Summary.FilesDeleted,
			SpaceFreed:   result.Summary.SpaceFreed,
		},
	}
	
	return plan, nil
}

// parseRenamingResponse parses Gemini's response into a RenamingPlan
func (g *GeminiAnalyzer) parseRenamingResponse(response string) (*RenamingPlan, error) {
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}") + 1
	
	if jsonStart == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no valid JSON found in response")
	}
	
	jsonStr := response[jsonStart:jsonEnd]
	
	var result struct {
		ID      string `json:"id"`
		Renames []struct {
			ID      string `json:"id"`
			OldName string `json:"oldName"`
			NewName string `json:"newName"`
			Reason  string `json:"reason"`
		} `json:"renames"`
		Summary struct {
			FilesRenamed int    `json:"filesRenamed"`
			Pattern      string `json:"pattern"`
		} `json:"summary"`
	}
	
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	
	renames := make([]Rename, len(result.Renames))
	for i, r := range result.Renames {
		renames[i] = Rename{
			ID:      r.ID,
			OldName: r.OldName,
			NewName: r.NewName,
			Reason:  r.Reason,
		}
	}
	
	plan := &RenamingPlan{
		ID:        result.ID,
		Timestamp: time.Now(),
		Renames:   renames,
		Summary: RenamingSummary{
			FilesRenamed: result.Summary.FilesRenamed,
			Pattern:      result.Summary.Pattern,
		},
	}
	
	return plan, nil
}