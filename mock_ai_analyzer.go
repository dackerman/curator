package curator

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// MockAIAnalyzer implements AIAnalyzer interface with rule-based logic
type MockAIAnalyzer struct{}

// NewMockAIAnalyzer creates a new mock AI analyzer
func NewMockAIAnalyzer() *MockAIAnalyzer {
	return &MockAIAnalyzer{}
}

// AnalyzeForReorganization implements AIAnalyzer.AnalyzeForReorganization
func (m *MockAIAnalyzer) AnalyzeForReorganization(files []FileInfo) (*ReorganizationPlan, error) {
	planID := fmt.Sprintf("reorg-%s", time.Now().Format("2006-01-02-150405"))
	
	var moves []Move
	moveID := 1
	
	// Analyze files and create reorganization moves based on simple rules
	documentExts := map[string]bool{
		".pdf": true, ".doc": true, ".docx": true, ".txt": true,
		".rtf": true, ".odt": true,
	}
	
	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".bmp": true, ".tiff": true, ".webp": true,
	}
	
	videoExts := map[string]bool{
		".mp4": true, ".avi": true, ".mov": true, ".wmv": true,
		".flv": true, ".mkv": true, ".webm": true,
	}
	
	// Count files by type
	var documentsCount, imagesCount, videosCount, othersCount int
	var needsDocumentsFolder, needsMediaFolder bool
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		ext := strings.ToLower(filepath.Ext(file.Name()))
		dir := filepath.Dir(file.Path())
		
		if documentExts[ext] {
			documentsCount++
			if dir != "/Documents" && dir != "Documents" {
				needsDocumentsFolder = true
			}
		} else if imageExts[ext] || videoExts[ext] {
			if imageExts[ext] {
				imagesCount++
			} else {
				videosCount++
			}
			if !strings.Contains(dir, "Media") && !strings.Contains(dir, "Photos") && !strings.Contains(dir, "Videos") {
				needsMediaFolder = true
			}
		} else {
			othersCount++
		}
	}
	
	// Create folder creation moves
	if needsDocumentsFolder && documentsCount > 0 {
		moves = append(moves, Move{
			ID:          fmt.Sprintf("move-%d", moveID),
			Source:      "",
			Destination: "/Documents",
			Reason:      "Create organized folder for document files",
			Type:        CreateFolder,
			FileCount:   0,
		})
		moveID++
	}
	
	if needsMediaFolder && (imagesCount > 0 || videosCount > 0) {
		moves = append(moves, Move{
			ID:          fmt.Sprintf("move-%d", moveID),
			Source:      "",
			Destination: "/Media",
			Reason:      "Create organized folder for media files",
			Type:        CreateFolder,
			FileCount:   0,
		})
		moveID++
		
		if imagesCount > 0 {
			moves = append(moves, Move{
				ID:          fmt.Sprintf("move-%d", moveID),
				Source:      "",
				Destination: "/Media/Photos",
				Reason:      "Separate photos from other media",
				Type:        CreateFolder,
				FileCount:   0,
			})
			moveID++
		}
		
		if videosCount > 0 {
			moves = append(moves, Move{
				ID:          fmt.Sprintf("move-%d", moveID),
				Source:      "",
				Destination: "/Media/Videos",
				Reason:      "Separate videos from other media",
				Type:        CreateFolder,
				FileCount:   0,
			})
			moveID++
		}
	}
	
	// Create file moves
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		ext := strings.ToLower(filepath.Ext(file.Name()))
		currentPath := file.Path()
		
		var destinationPath string
		var reason string
		
		if documentExts[ext] && !strings.Contains(currentPath, "/Documents") {
			destinationPath = "/Documents/" + file.Name()
			reason = "Group document files in dedicated Documents folder"
		} else if imageExts[ext] && !strings.Contains(currentPath, "/Media") && !strings.Contains(currentPath, "/Photos") {
			destinationPath = "/Media/Photos/" + file.Name()
			reason = "Organize photos in Media/Photos folder"
		} else if videoExts[ext] && !strings.Contains(currentPath, "/Media") && !strings.Contains(currentPath, "/Videos") {
			destinationPath = "/Media/Videos/" + file.Name()
			reason = "Organize videos in Media/Videos folder"
		}
		
		if destinationPath != "" {
			moves = append(moves, Move{
				ID:          fmt.Sprintf("move-%d", moveID),
				Source:      currentPath,
				Destination: destinationPath,
				Reason:      reason,
				Type:        FileMove,
				FileCount:   1,
			})
			moveID++
		}
	}
	
	// Calculate summary
	foldersCreated := 0
	filesMoved := 0
	for _, move := range moves {
		if move.Type == CreateFolder {
			foldersCreated++
		} else if move.Type == FileMove {
			filesMoved++
		}
	}
	
	summary := Summary{
		FoldersCreated:              foldersCreated,
		FilesMoved:                  filesMoved,
		FoldersMovedDeduplicated:    0,
		DepthReduction:              "15%",
		OrganizationImprovement:     fmt.Sprintf("%d%% of files will be in organized folders", (filesMoved*100)/(len(files)+1)),
	}
	
	return &ReorganizationPlan{
		ID:        planID,
		Timestamp: time.Now(),
		Moves:     moves,
		Summary:   summary,
		Rationale: "Organized files by type into logical folder structure for easier navigation and management",
	}, nil
}

// AnalyzeForDuplicates implements AIAnalyzer.AnalyzeForDuplicates
func (m *MockAIAnalyzer) AnalyzeForDuplicates(files []FileInfo) (*DuplicationReport, error) {
	duplicateGroups := make(map[string][]string)
	
	// Group files by hash
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		hash := file.Hash()
		if hash != "" {
			duplicateGroups[hash] = append(duplicateGroups[hash], file.Path())
		}
	}
	
	// Filter to only groups with duplicates
	var duplicates []DuplicateGroup
	totalDuplicates := 0
	var spaceSaved int64
	
	for hash, paths := range duplicateGroups {
		if len(paths) > 1 {
			// Find file size from any of the duplicates
			var size int64
			for _, file := range files {
				if file.Hash() == hash {
					size = file.Size()
					break
				}
			}
			
			duplicates = append(duplicates, DuplicateGroup{
				Hash:  hash,
				Files: paths,
				Size:  size,
			})
			
			totalDuplicates += len(paths) - 1 // All but one are duplicates
			spaceSaved += size * int64(len(paths)-1)
		}
	}
	
	return &DuplicationReport{
		ID:        fmt.Sprintf("dup-%s", time.Now().Format("2006-01-02-150405")),
		Timestamp: time.Now(),
		Duplicates: duplicates,
		Summary: DuplicationSummary{
			TotalDuplicates: totalDuplicates,
			SpaceSaved:      spaceSaved,
		},
	}, nil
}

// AnalyzeForCleanup implements AIAnalyzer.AnalyzeForCleanup
func (m *MockAIAnalyzer) AnalyzeForCleanup(files []FileInfo) (*CleanupPlan, error) {
	var deletions []Deletion
	deletionID := 1
	
	// Simple cleanup rules: temporary files, cache files, empty files
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		name := strings.ToLower(file.Name())
		var shouldDelete bool
		var reason string
		
		if strings.HasPrefix(name, "tmp") || strings.HasPrefix(name, "temp") {
			shouldDelete = true
			reason = "Temporary file that can be safely deleted"
		} else if strings.Contains(name, "cache") {
			shouldDelete = true
			reason = "Cache file that can be regenerated"
		} else if file.Size() == 0 {
			shouldDelete = true
			reason = "Empty file with no content"
		} else if strings.HasSuffix(name, ".log") && file.Size() > 1024*1024*10 { // 10MB
			shouldDelete = true
			reason = "Large log file that may be outdated"
		}
		
		if shouldDelete {
			deletions = append(deletions, Deletion{
				ID:     fmt.Sprintf("del-%d", deletionID),
				Path:   file.Path(),
				Reason: reason,
				Size:   file.Size(),
			})
			deletionID++
		}
	}
	
	var totalSize int64
	for _, deletion := range deletions {
		totalSize += deletion.Size
	}
	
	return &CleanupPlan{
		ID:        fmt.Sprintf("cleanup-%s", time.Now().Format("2006-01-02-150405")),
		Timestamp: time.Now(),
		Deletions: deletions,
		Summary: CleanupSummary{
			FilesDeleted: len(deletions),
			SpaceFreed:   totalSize,
		},
	}, nil
}

// AnalyzeForRenaming implements AIAnalyzer.AnalyzeForRenaming
func (m *MockAIAnalyzer) AnalyzeForRenaming(files []FileInfo) (*RenamingPlan, error) {
	var renames []Rename
	renameID := 1
	
	// Simple renaming rules: normalize spaces, remove special characters
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
				Reason:  "Normalize filename to follow consistent naming convention",
			})
			renameID++
		}
	}
	
	return &RenamingPlan{
		ID:        fmt.Sprintf("rename-%s", time.Now().Format("2006-01-02-150405")),
		Timestamp: time.Now(),
		Renames:   renames,
		Summary: RenamingSummary{
			FilesRenamed: len(renames),
			Pattern:      "consistent-naming",
		},
	}, nil
}

// normalizeFileName applies consistent naming rules
func normalizeFileName(name string) string {
	// Replace spaces with underscores
	name = strings.ReplaceAll(name, " ", "_")
	
	// Remove multiple underscores
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	
	// Convert to lowercase (but preserve extension case)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return strings.ToLower(base) + ext
}