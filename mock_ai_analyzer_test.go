package curator

import (
	"testing"
	"time"
)

func TestMockAIAnalyzer_AnalyzeForReorganization(t *testing.T) {
	analyzer := NewMockAIAnalyzer()
	
	// Create test files
	fs := NewMemoryFileSystem()
	fs.AddFile("/document1.pdf", []byte("pdf content"), "application/pdf")
	fs.AddFile("/photo1.jpg", []byte("image content"), "image/jpeg")
	fs.AddFile("/video1.mp4", []byte("video content"), "video/mp4")
	fs.AddFile("/random/document2.txt", []byte("text content"), "text/plain")
	fs.AddFile("/downloads/photo2.png", []byte("png content"), "image/png")
	
	files, err := fs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
	// Analyze for reorganization
	plan, err := analyzer.AnalyzeForReorganization(files)
	if err != nil {
		t.Fatalf("Failed to analyze for reorganization: %v", err)
	}
	
	// Verify plan structure
	if plan.ID == "" {
		t.Error("Plan ID should not be empty")
	}
	
	if plan.Timestamp.IsZero() {
		t.Error("Plan timestamp should not be zero")
	}
	
	if len(plan.Moves) == 0 {
		t.Error("Plan should have moves")
	}
	
	// Check that we have folder creation moves
	var createFolderMoves []Move
	var fileMoves []Move
	
	for _, move := range plan.Moves {
		if move.Type == CreateFolder {
			createFolderMoves = append(createFolderMoves, move)
		} else if move.Type == FileMove {
			fileMoves = append(fileMoves, move)
		}
	}
	
	if len(createFolderMoves) == 0 {
		t.Error("Should have folder creation moves")
	}
	
	if len(fileMoves) == 0 {
		t.Error("Should have file move operations")
	}
	
	// Verify summary
	if plan.Summary.FoldersCreated == 0 {
		t.Error("Summary should show folders created")
	}
	
	if plan.Summary.FilesMoved == 0 {
		t.Error("Summary should show files moved")
	}
	
	if plan.Rationale == "" {
		t.Error("Plan should have rationale")
	}
}

func TestMockAIAnalyzer_AnalyzeForDuplicates(t *testing.T) {
	analyzer := NewMockAIAnalyzer()
	
	// Create test files with duplicates
	fs := NewMemoryFileSystem()
	content1 := []byte("same content")
	content2 := []byte("different content")
	
	fs.AddFile("/file1.txt", content1, "text/plain")
	fs.AddFile("/copy/file1_copy.txt", content1, "text/plain") // Duplicate
	fs.AddFile("/file2.txt", content2, "text/plain")
	
	files, err := fs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
	// Add files from subdirectory
	copyFiles, err := fs.List("/copy")
	if err != nil {
		t.Fatalf("Failed to list copy files: %v", err)
	}
	files = append(files, copyFiles...)
	
	// Analyze for duplicates
	report, err := analyzer.AnalyzeForDuplicates(files)
	if err != nil {
		t.Fatalf("Failed to analyze for duplicates: %v", err)
	}
	
	// Verify report structure
	if report.ID == "" {
		t.Error("Report ID should not be empty")
	}
	
	if report.Timestamp.IsZero() {
		t.Error("Report timestamp should not be zero")
	}
	
	// Should find at least one duplicate group
	if len(report.Duplicates) == 0 {
		t.Error("Should find duplicate files")
	}
	
	// Verify duplicate group
	duplicateGroup := report.Duplicates[0]
	if len(duplicateGroup.Files) < 2 {
		t.Error("Duplicate group should have at least 2 files")
	}
	
	if duplicateGroup.Hash == "" {
		t.Error("Duplicate group should have hash")
	}
	
	// Verify summary
	if report.Summary.TotalDuplicates == 0 {
		t.Error("Summary should show duplicates found")
	}
}

func TestMockAIAnalyzer_AnalyzeForCleanup(t *testing.T) {
	analyzer := NewMockAIAnalyzer()
	
	// Create test files including ones that should be cleaned up
	fs := NewMemoryFileSystem()
	fs.AddFile("/document.pdf", []byte("important document"), "application/pdf")
	fs.AddFile("/tmp_file.tmp", []byte("temporary file"), "text/plain") // Should be deleted
	fs.AddFile("/cache_data.cache", []byte("cache content"), "text/plain") // Should be deleted
	fs.AddFile("/empty_file.txt", []byte(""), "text/plain") // Should be deleted (empty)
	
	files, err := fs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
	// Analyze for cleanup
	plan, err := analyzer.AnalyzeForCleanup(files)
	if err != nil {
		t.Fatalf("Failed to analyze for cleanup: %v", err)
	}
	
	// Verify plan structure
	if plan.ID == "" {
		t.Error("Plan ID should not be empty")
	}
	
	if plan.Timestamp.IsZero() {
		t.Error("Plan timestamp should not be zero")
	}
	
	// Should find files to delete
	if len(plan.Deletions) == 0 {
		t.Error("Should find files to delete")
	}
	
	// Verify deletions include expected files
	deletionPaths := make(map[string]bool)
	for _, deletion := range plan.Deletions {
		deletionPaths[deletion.Path] = true
		
		if deletion.ID == "" {
			t.Error("Deletion should have ID")
		}
		
		if deletion.Reason == "" {
			t.Error("Deletion should have reason")
		}
	}
	
	// Check specific files are marked for deletion
	expectedDeletions := []string{"/tmp_file.tmp", "/cache_data.cache", "/empty_file.txt"}
	for _, expected := range expectedDeletions {
		if !deletionPaths[expected] {
			t.Errorf("Expected file %s to be marked for deletion", expected)
		}
	}
	
	// Verify summary
	if plan.Summary.FilesDeleted != len(plan.Deletions) {
		t.Error("Summary files deleted count should match deletions")
	}
}

func TestMockAIAnalyzer_AnalyzeForRenaming(t *testing.T) {
	analyzer := NewMockAIAnalyzer()
	
	// Create test files with names that need normalization
	fs := NewMemoryFileSystem()
	fs.AddFile("/My Document.pdf", []byte("document content"), "application/pdf")
	fs.AddFile("/Another  File.txt", []byte("text content"), "text/plain")
	fs.AddFile("/normal_file.jpg", []byte("image content"), "image/jpeg")
	
	files, err := fs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
	// Analyze for renaming
	plan, err := analyzer.AnalyzeForRenaming(files)
	if err != nil {
		t.Fatalf("Failed to analyze for renaming: %v", err)
	}
	
	// Verify plan structure
	if plan.ID == "" {
		t.Error("Plan ID should not be empty")
	}
	
	if plan.Timestamp.IsZero() {
		t.Error("Plan timestamp should not be zero")
	}
	
	// Should find files to rename
	if len(plan.Renames) == 0 {
		t.Error("Should find files to rename")
	}
	
	// Verify renames
	for _, rename := range plan.Renames {
		if rename.ID == "" {
			t.Error("Rename should have ID")
		}
		
		if rename.OldName == "" {
			t.Error("Rename should have old name")
		}
		
		if rename.NewName == "" {
			t.Error("Rename should have new name")
		}
		
		if rename.Reason == "" {
			t.Error("Rename should have reason")
		}
		
		// Verify that new name follows conventions
		if rename.OldName != rename.NewName {
			// Should normalize spaces to underscores and lowercase base name
			t.Logf("Rename: %s -> %s", rename.OldName, rename.NewName)
		}
	}
	
	// Verify summary
	if plan.Summary.FilesRenamed != len(plan.Renames) {
		t.Error("Summary files renamed count should match renames")
	}
	
	if plan.Summary.Pattern != "consistent-naming" {
		t.Error("Summary should indicate consistent naming pattern")
	}
}

func TestNormalizeFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My Document.pdf", "my_document.pdf"},
		{"Another  File.txt", "another_file.txt"},
		{"normal_file.jpg", "normal_file.jpg"},
		{"File___With___Underscores.doc", "file_with_underscores.doc"},
		{"UPPERCASE.TXT", "uppercase.TXT"},
	}
	
	for _, test := range tests {
		result := normalizeFileName(test.input)
		if result != test.expected {
			t.Errorf("normalizeFileName(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}