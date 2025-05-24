package curator

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// MockAIAnalyzer implements AIAnalyzer interface with simple heuristics
type MockAIAnalyzer struct{}

// NewMockAIAnalyzer creates a new mock AI analyzer
func NewMockAIAnalyzer() *MockAIAnalyzer {
	return &MockAIAnalyzer{}
}

// AnalyzeForReorganization implements AIAnalyzer.AnalyzeForReorganization
func (m *MockAIAnalyzer) AnalyzeForReorganization(files []FileInfo) (*ReorganizationPlan, error) {
	planID := fmt.Sprintf("reorg-%d", time.Now().Unix())
	
	var moves []Move
	moveID := 1
	
	// Simple heuristics for organization:
	// 1. Group files by extension in folders
	// 2. Move documents to a Documents folder
	// 3. Move images to an Images folder
	// 4. Move downloads to proper categories
	
	filesByExt := make(map[string][]FileInfo)
	rootFiles := make([]FileInfo, 0)
	
	// Categorize files
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext == "" {
			ext = "no-extension"
		}
		
		// Skip files already in organized folders
		if isInOrganizedFolder(file.Path()) {
			continue
		}
		
		filesByExt[ext] = append(filesByExt[ext], file)
		rootFiles = append(rootFiles, file)
	}
	
	// Create folder structure moves
	docExts := []string{".pdf", ".doc", ".docx", ".txt", ".md", ".rtf"}
	imgExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".svg"}
	videoExts := []string{".mp4", ".avi", ".mov", ".mkv", ".wmv", ".flv"}
	audioExts := []string{".mp3", ".wav", ".flac", ".aac", ".ogg"}
	archiveExts := []string{".zip", ".rar", ".7z", ".tar", ".gz"}
	codeExts := []string{".go", ".js", ".py", ".java", ".cpp", ".c", ".h"}
	
	folderMappings := map[string]string{
		"documents": "Documents",
		"images":    "Images", 
		"videos":    "Videos",
		"audio":     "Audio",
		"archives":  "Archives",
		"code":      "Code",
		"other":     "Other",
	}
	
	// Create folders first
	for _, folder := range folderMappings {
		moves = append(moves, Move{
			ID:          fmt.Sprintf("move-%d", moveID),
			Source:      "",
			Destination: folder,
			Reason:      fmt.Sprintf("Create %s folder for better organization", folder),
			Type:        CreateFolder,
			FileCount:   0,
		})
		moveID++
	}
	
	// Move files to appropriate folders
	for ext, extFiles := range filesByExt {
		if len(extFiles) == 0 {
			continue
		}
		
		var targetFolder string
		var category string
		
		switch {
		case contains(docExts, ext):
			targetFolder = "Documents"
			category = "documents"
		case contains(imgExts, ext):
			targetFolder = "Images"
			category = "images"
		case contains(videoExts, ext):
			targetFolder = "Videos"
			category = "videos"
		case contains(audioExts, ext):
			targetFolder = "Audio"
			category = "audio"
		case contains(archiveExts, ext):
			targetFolder = "Archives"
			category = "archives"
		case contains(codeExts, ext):
			targetFolder = "Code"
			category = "code"
		default:
			targetFolder = "Other"
			category = "other"
		}
		
		for _, file := range extFiles {
			moves = append(moves, Move{
				ID:          fmt.Sprintf("move-%d", moveID),
				Source:      file.Path(),
				Destination: filepath.Join(targetFolder, file.Name()),
				Reason:      fmt.Sprintf("Move %s file to %s folder", category, targetFolder),
				Type:        FileMove,
				FileCount:   1,
			})
			moveID++
		}
	}
	
	// Calculate summary
	summary := Summary{
		FoldersCreated:              len(folderMappings),
		FilesMoved:                  len(rootFiles),
		FoldersMovedDeduplicated:    0,
		DepthReduction:              "0%",
		OrganizationImprovement:     fmt.Sprintf("%d%% of files will be organized by type", (len(rootFiles)*100)/(len(files)+1)),
	}
	
	plan := &ReorganizationPlan{
		ID:        planID,
		Timestamp: time.Now(),
		Moves:     moves,
		Summary:   summary,
		Rationale: "Organize files by type into dedicated folders to improve findability and reduce clutter in root directory",
	}
	
	return plan, nil
}

// AnalyzeForDuplicates implements AIAnalyzer.AnalyzeForDuplicates
func (m *MockAIAnalyzer) AnalyzeForDuplicates(files []FileInfo) (*DuplicationReport, error) {
	hashToFiles := make(map[string][]string)
	
	// Group files by hash
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		hash := file.Hash()
		if hash != "" {
			hashToFiles[hash] = append(hashToFiles[hash], file.Path())
		}
	}
	
	var duplicates []DuplicateGroup
	var totalDuplicates int
	var spaceSaved int64
	
	// Find duplicate groups (more than 1 file with same hash)
	for hash, filePaths := range hashToFiles {
		if len(filePaths) > 1 {
			// Calculate size from first file
			var size int64
			for _, file := range files {
				if file.Hash() == hash {
					size = file.Size()
					break
				}
			}
			
			duplicates = append(duplicates, DuplicateGroup{
				Hash:  hash,
				Files: filePaths,
				Size:  size,
			})
			
			totalDuplicates += len(filePaths) - 1
			spaceSaved += size * int64(len(filePaths)-1)
		}
	}
	
	report := &DuplicationReport{
		ID:        fmt.Sprintf("dup-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Duplicates: duplicates,
		Summary: DuplicationSummary{
			TotalDuplicates: totalDuplicates,
			SpaceSaved:      spaceSaved,
		},
	}
	
	return report, nil
}

// AnalyzeForCleanup implements AIAnalyzer.AnalyzeForCleanup
func (m *MockAIAnalyzer) AnalyzeForCleanup(files []FileInfo) (*CleanupPlan, error) {
	var deletions []Deletion
	deletionID := 1
	var totalSize int64
	
	// Simple cleanup heuristics:
	// 1. Delete empty files
	// 2. Delete temporary files (.tmp, .temp)
	// 3. Delete cache files
	// 4. Delete old log files
	
	junkPatterns := []string{
		".tmp", ".temp", ".cache", ".log", ".bak", "~",
	}
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		shouldDelete := false
		reason := ""
		
		// Check if empty file
		if file.Size() == 0 {
			shouldDelete = true
			reason = "Empty file taking up space"
		}
		
		// Check if matches junk patterns
		name := strings.ToLower(file.Name())
		for _, pattern := range junkPatterns {
			if strings.HasSuffix(name, pattern) || strings.Contains(name, pattern) {
				shouldDelete = true
				reason = fmt.Sprintf("Temporary/junk file matching pattern '%s'", pattern)
				break
			}
		}
		
		if shouldDelete {
			deletions = append(deletions, Deletion{
				ID:     fmt.Sprintf("del-%d", deletionID),
				Path:   file.Path(),
				Reason: reason,
				Size:   file.Size(),
			})
			totalSize += file.Size()
			deletionID++
		}
	}
	
	plan := &CleanupPlan{
		ID:        fmt.Sprintf("cleanup-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Deletions: deletions,
		Summary: CleanupSummary{
			FilesDeleted: len(deletions),
			SpaceFreed:   totalSize,
		},
	}
	
	return plan, nil
}

// AnalyzeForRenaming implements AIAnalyzer.AnalyzeForRenaming
func (m *MockAIAnalyzer) AnalyzeForRenaming(files []FileInfo) (*RenamingPlan, error) {
	var renames []Rename
	renameID := 1
	
	// Simple renaming heuristics:
	// 1. Remove spaces and replace with underscores
	// 2. Convert to lowercase
	// 3. Remove special characters
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		oldName := file.Name()
		newName := normalizeFileName(oldName)
		
		if oldName != newName {
			renames = append(renames, Rename{
				ID:      fmt.Sprintf("rename-%d", renameID),
				OldName: oldName,
				NewName: newName,
				Reason:  "Standardize filename to consistent naming convention",
			})
			renameID++
		}
	}
	
	plan := &RenamingPlan{
		ID:        fmt.Sprintf("rename-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Renames:   renames,
		Summary: RenamingSummary{
			FilesRenamed: len(renames),
			Pattern:      "lowercase_with_underscores",
		},
	}
	
	return plan, nil
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func isInOrganizedFolder(path string) bool {
	organizedFolders := []string{"Documents", "Images", "Videos", "Audio", "Archives", "Code", "Other"}
	
	for _, folder := range organizedFolders {
		if strings.HasPrefix(path, folder+"/") {
			return true
		}
	}
	return false
}

func normalizeFileName(name string) string {
	// Keep the extension
	ext := filepath.Ext(name)
	nameWithoutExt := strings.TrimSuffix(name, ext)
	
	// Convert to lowercase
	normalized := strings.ToLower(nameWithoutExt)
	
	// Replace spaces with underscores
	normalized = strings.ReplaceAll(normalized, " ", "_")
	
	// Remove special characters except underscores and hyphens
	var result strings.Builder
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		}
	}
	
	return result.String() + ext
}