package curator

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalFileSystem_NewLocalFileSystem(t *testing.T) {
	// Test with valid directory
	tmpDir, err := os.MkdirTemp("", "curator-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	lfs, err := NewLocalFileSystem(tmpDir)
	if err != nil {
		t.Errorf("Expected success with valid directory, got error: %v", err)
	}
	
	if lfs.GetRootPath() != tmpDir {
		t.Errorf("Expected root path %s, got %s", tmpDir, lfs.GetRootPath())
	}
	
	// Test with non-existent directory
	_, err = NewLocalFileSystem("/non/existent/path")
	if err == nil {
		t.Error("Expected error with non-existent directory")
	}
	
	// Test with file instead of directory
	tmpFile, err := os.CreateTemp("", "curator-test-file-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()
	
	_, err = NewLocalFileSystem(tmpFile.Name())
	if err == nil {
		t.Error("Expected error when root path is a file")
	}
}

func TestLocalFileSystem_CreateFolderAndList(t *testing.T) {
	tmpDir, lfs := setupTestFS(t)
	defer os.RemoveAll(tmpDir)
	
	// Create a folder
	err := lfs.CreateFolder("/testdir")
	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}
	
	// Verify folder exists
	exists, err := lfs.Exists("/testdir")
	if err != nil {
		t.Fatalf("Failed to check if folder exists: %v", err)
	}
	if !exists {
		t.Error("Folder should exist after creation")
	}
	
	// List contents of root
	files, err := lfs.List("/")
	if err != nil {
		t.Fatalf("Failed to list root directory: %v", err)
	}
	
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
	
	if files[0].Name() != "testdir" {
		t.Errorf("Expected file name 'testdir', got %s", files[0].Name())
	}
	
	if !files[0].IsDir() {
		t.Error("Expected directory, got file")
	}
	
	if files[0].Path() != "/testdir" {
		t.Errorf("Expected path '/testdir', got %s", files[0].Path())
	}
}

func TestLocalFileSystem_CreateFileAndRead(t *testing.T) {
	tmpDir, lfs := setupTestFS(t)
	defer os.RemoveAll(tmpDir)
	
	// Create a test file directly on disk
	testContent := "Hello, World!"
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// List and verify
	files, err := lfs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
	
	file := files[0]
	if file.Name() != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got %s", file.Name())
	}
	
	if file.IsDir() {
		t.Error("Expected file, got directory")
	}
	
	if file.Size() != int64(len(testContent)) {
		t.Errorf("Expected size %d, got %d", len(testContent), file.Size())
	}
	
	// Test reading
	reader, err := lfs.Read("/test.txt")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}
	
	if string(content) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(content))
	}
	
	// Close the reader if it's a file
	if closer, ok := reader.(io.Closer); ok {
		closer.Close()
	}
}

func TestLocalFileSystem_Move(t *testing.T) {
	tmpDir, lfs := setupTestFS(t)
	defer os.RemoveAll(tmpDir)
	
	// Create source file
	testContent := "Test content"
	srcFile := filepath.Join(tmpDir, "source.txt")
	err := os.WriteFile(srcFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	// Create destination directory
	err = lfs.CreateFolder("/dest")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	
	// Move file
	err = lfs.Move("/source.txt", "/dest/moved.txt")
	if err != nil {
		t.Fatalf("Failed to move file: %v", err)
	}
	
	// Verify source no longer exists
	exists, err := lfs.Exists("/source.txt")
	if err != nil {
		t.Fatalf("Failed to check source existence: %v", err)
	}
	if exists {
		t.Error("Source file should not exist after move")
	}
	
	// Verify destination exists
	exists, err = lfs.Exists("/dest/moved.txt")
	if err != nil {
		t.Fatalf("Failed to check destination existence: %v", err)
	}
	if !exists {
		t.Error("Destination file should exist after move")
	}
	
	// Verify content is preserved
	reader, err := lfs.Read("/dest/moved.txt")
	if err != nil {
		t.Fatalf("Failed to read moved file: %v", err)
	}
	
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}
	
	if string(content) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(content))
	}
	
	if closer, ok := reader.(io.Closer); ok {
		closer.Close()
	}
}

func TestLocalFileSystem_Delete(t *testing.T) {
	tmpDir, lfs := setupTestFS(t)
	defer os.RemoveAll(tmpDir)
	
	// Create test file
	testFile := filepath.Join(tmpDir, "delete_me.txt")
	err := os.WriteFile(testFile, []byte("delete me"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Verify file exists
	exists, err := lfs.Exists("/delete_me.txt")
	if err != nil {
		t.Fatalf("Failed to check file existence: %v", err)
	}
	if !exists {
		t.Error("File should exist before deletion")
	}
	
	// Delete file
	err = lfs.Delete("/delete_me.txt")
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}
	
	// Verify file no longer exists
	exists, err = lfs.Exists("/delete_me.txt")
	if err != nil {
		t.Fatalf("Failed to check file existence after deletion: %v", err)
	}
	if exists {
		t.Error("File should not exist after deletion")
	}
}

func TestLocalFileSystem_Hash(t *testing.T) {
	tmpDir, lfs := setupTestFS(t)
	defer os.RemoveAll(tmpDir)
	
	// Create two files with same content
	testContent := "Same content for hashing"
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	
	err := os.WriteFile(file1, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	
	err = os.WriteFile(file2, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}
	
	// Create file with different content
	file3 := filepath.Join(tmpDir, "file3.txt")
	err = os.WriteFile(file3, []byte("Different content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file3: %v", err)
	}
	
	// List files and check hashes
	files, err := lfs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
	if len(files) != 3 {
		t.Fatalf("Expected 3 files, got %d", len(files))
	}
	
	// Find files by name
	var hash1, hash2, hash3 string
	for _, file := range files {
		switch file.Name() {
		case "file1.txt":
			hash1 = file.Hash()
		case "file2.txt":
			hash2 = file.Hash()
		case "file3.txt":
			hash3 = file.Hash()
		}
	}
	
	// Files with same content should have same hash
	if hash1 != hash2 {
		t.Errorf("Files with same content should have same hash: %s != %s", hash1, hash2)
	}
	
	// File with different content should have different hash
	if hash1 == hash3 {
		t.Errorf("Files with different content should have different hash: %s == %s", hash1, hash3)
	}
	
	// Hashes should not be empty
	if hash1 == "" || hash2 == "" || hash3 == "" {
		t.Error("File hashes should not be empty")
	}
}

func TestLocalFileSystem_MimeType(t *testing.T) {
	tmpDir, lfs := setupTestFS(t)
	defer os.RemoveAll(tmpDir)
	
	// Create files with different extensions
	files := map[string]string{
		"text.txt":    "text/plain",
		"image.jpg":   "image/jpeg",
		"doc.pdf":     "application/pdf",
		"data.json":   "application/json",
		"style.css":   "text/css",
	}
	
	for filename := range files {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
	}
	
	// List files and check MIME types
	fileInfos, err := lfs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
	for _, fileInfo := range fileInfos {
		expected, exists := files[fileInfo.Name()]
		if !exists {
			continue
		}
		
		actualMime := fileInfo.MimeType()
		
		// Some MIME types might include charset, so check if it starts with expected
		if !strings.HasPrefix(actualMime, expected) {
			t.Errorf("File %s: expected MIME type to start with '%s', got '%s'", 
				fileInfo.Name(), expected, actualMime)
		}
	}
}

func TestLocalFileSystem_PathSecurity(t *testing.T) {
	tmpDir, lfs := setupTestFS(t)
	defer os.RemoveAll(tmpDir)
	
	// Try to access files outside root directory
	_, err := lfs.List("../../")
	if err == nil {
		t.Error("Should not be able to access files outside root directory")
	}
	
	// Try to create folder outside root
	err = lfs.CreateFolder("../outside")
	if err == nil {
		t.Error("Should not be able to create folder outside root directory")
	}
	
	// Try to move file outside root
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	err = lfs.Move("/test.txt", "../outside.txt")
	if err == nil {
		t.Error("Should not be able to move file outside root directory")
	}
}

func TestLocalFileSystem_ErrorHandling(t *testing.T) {
	tmpDir, lfs := setupTestFS(t)
	defer os.RemoveAll(tmpDir)
	
	// Test reading non-existent file
	_, err := lfs.Read("/nonexistent.txt")
	if err == nil {
		t.Error("Should return error when reading non-existent file")
	}
	
	// Test moving non-existent file
	err = lfs.Move("/nonexistent.txt", "/dest.txt")
	if err == nil {
		t.Error("Should return error when moving non-existent file")
	}
	
	// Test deleting non-existent file
	err = lfs.Delete("/nonexistent.txt")
	if err == nil {
		t.Error("Should return error when deleting non-existent file")
	}
	
	// Test moving to existing destination
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	
	err = os.WriteFile(file1, []byte("content1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	
	err = os.WriteFile(file2, []byte("content2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}
	
	err = lfs.Move("/file1.txt", "/file2.txt")
	if err == nil {
		t.Error("Should return error when moving to existing destination")
	}
}

// Helper function to set up test filesystem
func setupTestFS(t *testing.T) (string, *LocalFileSystem) {
	tmpDir, err := os.MkdirTemp("", "curator-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	lfs, err := NewLocalFileSystem(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create LocalFileSystem: %v", err)
	}
	
	return tmpDir, lfs
}