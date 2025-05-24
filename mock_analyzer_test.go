package curator

import (
	"testing"
)

func TestMockAIAnalyzer_AnalyzeForReorganization(t *testing.T) {
	analyzer := NewMockAIAnalyzer()
	
	// Create test files
	fs := NewMemoryFileSystem()
	fs.AddFile("/document.pdf", []byte("pdf content"), "application/pdf")
	fs.AddFile("/image.jpg", []byte("image content"), "image/jpeg")
	fs.AddFile("/video.mp4", []byte("video content"), "video/mp4")
	fs.AddFile("/script.go", []byte("package main"), "text/plain")
	
	files, err := fs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
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
	
	// Check that folders are created
	folderCreations := 0
	fileMoves := 0
	
	for _, move := range plan.Moves {
		switch move.Type {
		case CreateFolder:
			folderCreations++
		case FileMove:
			fileMoves++
		}
	}
	
	if folderCreations == 0 {
		t.Error("Should create folders for organization")
	}
	
	if fileMoves == 0 {
		t.Error("Should move files into organized folders")
	}
	
	// Verify summary
	if plan.Summary.FoldersCreated == 0 {
		t.Error("Summary should show folders created")
	}
	
	if plan.Summary.FilesMoved == 0 {
		t.Error("Summary should show files moved")
	}
}

func TestMockAIAnalyzer_AnalyzeForDuplicates(t *testing.T) {
	analyzer := NewMockAIAnalyzer()
	
	// Create test files with duplicates
	fs := NewMemoryFileSystem()
	fs.AddFile("/file1.txt", []byte("same content"), "text/plain")
	fs.AddFile("/file2.txt", []byte("same content"), "text/plain")
	fs.AddFile("/file3.txt", []byte("different content"), "text/plain")
	
	files, err := fs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
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
	
	// Should find one duplicate group (file1.txt and file2.txt have same content)
	if len(report.Duplicates) != 1 {
		t.Errorf("Expected 1 duplicate group, got %d", len(report.Duplicates))
	}
	
	if len(report.Duplicates) > 0 {
		group := report.Duplicates[0]
		if len(group.Files) != 2 {
			t.Errorf("Expected 2 files in duplicate group, got %d", len(group.Files))
		}
	}
	
	// Check summary
	if report.Summary.TotalDuplicates != 1 {
		t.Errorf("Expected 1 total duplicate, got %d", report.Summary.TotalDuplicates)
	}
}

func TestMockAIAnalyzer_AnalyzeForCleanup(t *testing.T) {
	analyzer := NewMockAIAnalyzer()
	
	// Create test files with junk
	fs := NewMemoryFileSystem()
	fs.AddFile("/document.pdf", []byte("good file"), "application/pdf")
	fs.AddFile("/temp.tmp", []byte("temporary"), "text/plain")
	fs.AddFile("/empty.txt", []byte(""), "text/plain")
	fs.AddFile("/backup.bak", []byte("backup"), "text/plain")
	
	files, err := fs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
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
	
	// Should identify junk files (temp.tmp, empty.txt, backup.bak)
	expectedDeletions := 3
	if len(plan.Deletions) != expectedDeletions {
		t.Errorf("Expected %d deletions, got %d", expectedDeletions, len(plan.Deletions))
	}
	
	// Check that good file is not marked for deletion
	for _, deletion := range plan.Deletions {
		if deletion.Path == "/document.pdf" {
			t.Error("Good file should not be marked for deletion")
		}
	}
	
	// Verify summary
	if plan.Summary.FilesDeleted != len(plan.Deletions) {
		t.Error("Summary should match actual deletions")
	}
}

func TestMockAIAnalyzer_AnalyzeForRenaming(t *testing.T) {
	analyzer := NewMockAIAnalyzer()
	
	// Create test files with inconsistent names
	fs := NewMemoryFileSystem()
	fs.AddFile("/My Document.pdf", []byte("doc"), "application/pdf")
	fs.AddFile("/Photo With Spaces.jpg", []byte("photo"), "image/jpeg")
	fs.AddFile("/already_good.txt", []byte("text"), "text/plain")
	fs.AddFile("/file-with-hyphens.txt", []byte("text"), "text/plain")
	
	files, err := fs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
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
	
	// Should identify files that need renaming (those with spaces)
	renamedFound := false
	for _, rename := range plan.Renames {
		if rename.OldName == "My Document.pdf" && rename.NewName == "my_document.pdf" {
			renamedFound = true
		}
		if rename.OldName == "already_good.txt" {
			t.Error("File with good name should not be renamed")
		}
	}
	
	if !renamedFound {
		t.Error("Should rename files with spaces")
	}
	
	// Verify summary
	if plan.Summary.Pattern == "" {
		t.Error("Summary should include naming pattern")
	}
}