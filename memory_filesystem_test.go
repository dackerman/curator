package curator

import (
	"io"
	"testing"
)

func TestMemoryFileSystem_AddFileAndList(t *testing.T) {
	mfs := NewMemoryFileSystem()
	
	// Add a file
	content := []byte("test content")
	mfs.AddFile("/test.txt", content, "text/plain")
	
	// List files in root
	files, err := mfs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}
	
	file := files[0]
	if file.Name() != "test.txt" {
		t.Errorf("Expected name 'test.txt', got '%s'", file.Name())
	}
	
	if file.Path() != "/test.txt" {
		t.Errorf("Expected path '/test.txt', got '%s'", file.Path())
	}
	
	if file.IsDir() {
		t.Error("File should not be a directory")
	}
	
	if file.Size() != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), file.Size())
	}
	
	if file.MimeType() != "text/plain" {
		t.Errorf("Expected mime type 'text/plain', got '%s'", file.MimeType())
	}
}

func TestMemoryFileSystem_AddFolderAndList(t *testing.T) {
	mfs := NewMemoryFileSystem()
	
	// Add a folder
	mfs.AddFolder("/docs")
	
	// List files in root
	files, err := mfs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}
	
	folder := files[0]
	if folder.Name() != "docs" {
		t.Errorf("Expected name 'docs', got '%s'", folder.Name())
	}
	
	if !folder.IsDir() {
		t.Error("Folder should be a directory")
	}
}

func TestMemoryFileSystem_Read(t *testing.T) {
	mfs := NewMemoryFileSystem()
	
	content := []byte("hello world")
	mfs.AddFile("/hello.txt", content, "text/plain")
	
	reader, err := mfs.Read("/hello.txt")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	
	readContent, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}
	
	if string(readContent) != string(content) {
		t.Errorf("Expected content '%s', got '%s'", string(content), string(readContent))
	}
}

func TestMemoryFileSystem_Move(t *testing.T) {
	mfs := NewMemoryFileSystem()
	
	// Add a file
	content := []byte("test content")
	mfs.AddFile("/test.txt", content, "text/plain")
	
	// Move the file
	err := mfs.Move("/test.txt", "/moved/test.txt")
	if err != nil {
		t.Fatalf("Failed to move file: %v", err)
	}
	
	// Check that original doesn't exist
	exists, err := mfs.Exists("/test.txt")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("Original file should not exist after move")
	}
	
	// Check that new location exists
	exists, err = mfs.Exists("/moved/test.txt")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Moved file should exist at new location")
	}
	
	// Check that parent directory was created
	exists, err = mfs.Exists("/moved")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Parent directory should be created automatically")
	}
	
	// Verify content is preserved
	reader, err := mfs.Read("/moved/test.txt")
	if err != nil {
		t.Fatalf("Failed to read moved file: %v", err)
	}
	
	readContent, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}
	
	if string(readContent) != string(content) {
		t.Errorf("Content should be preserved after move")
	}
}

func TestMemoryFileSystem_Delete(t *testing.T) {
	mfs := NewMemoryFileSystem()
	
	// Add a file
	mfs.AddFile("/test.txt", []byte("content"), "text/plain")
	
	// Delete the file
	err := mfs.Delete("/test.txt")
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}
	
	// Check that it doesn't exist
	exists, err := mfs.Exists("/test.txt")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("File should not exist after deletion")
	}
}

func TestMemoryFileSystem_Hash(t *testing.T) {
	mfs := NewMemoryFileSystem()
	
	// Add two files with same content
	content := []byte("identical content")
	mfs.AddFile("/file1.txt", content, "text/plain")
	mfs.AddFile("/file2.txt", content, "text/plain")
	
	files, err := mfs.List("/")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	
	if len(files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(files))
	}
	
	// Files with same content should have same hash
	hash1 := files[0].Hash()
	hash2 := files[1].Hash()
	
	if hash1 != hash2 {
		t.Errorf("Files with identical content should have same hash")
	}
	
	if hash1 == "" {
		t.Error("Hash should not be empty for files")
	}
}

func TestMemoryFileSystem_NestedDirectories(t *testing.T) {
	mfs := NewMemoryFileSystem()
	
	// Add files in nested structure
	mfs.AddFile("/docs/personal/notes.txt", []byte("notes"), "text/plain")
	mfs.AddFile("/docs/work/report.pdf", []byte("pdf content"), "application/pdf")
	
	// List root directory
	rootFiles, err := mfs.List("/")
	if err != nil {
		t.Fatalf("Failed to list root: %v", err)
	}
	
	if len(rootFiles) != 1 {
		t.Fatalf("Expected 1 item in root, got %d", len(rootFiles))
	}
	
	if rootFiles[0].Name() != "docs" || !rootFiles[0].IsDir() {
		t.Error("Root should contain 'docs' directory")
	}
	
	// List docs directory
	docsFiles, err := mfs.List("/docs")
	if err != nil {
		t.Fatalf("Failed to list docs: %v", err)
	}
	
	if len(docsFiles) != 2 {
		t.Fatalf("Expected 2 items in docs, got %d", len(docsFiles))
	}
	
	// Check that we have personal and work directories
	var hasPersonal, hasWork bool
	for _, file := range docsFiles {
		if file.Name() == "personal" && file.IsDir() {
			hasPersonal = true
		}
		if file.Name() == "work" && file.IsDir() {
			hasWork = true
		}
	}
	
	if !hasPersonal {
		t.Error("Should have personal directory")
	}
	if !hasWork {
		t.Error("Should have work directory")
	}
}