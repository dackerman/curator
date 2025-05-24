package curator

import (
	"os"
	"testing"
	"time"
)

func TestDefaultGoogleDriveConfig(t *testing.T) {
	config := DefaultGoogleDriveConfig()
	
	if config.ApplicationName == "" {
		t.Error("Expected default application name to be set")
	}
	
	if config.ServiceAccountKey != "" {
		t.Error("Expected service account key to be empty by default")
	}
	
	if config.RootFolderID != "" {
		t.Error("Expected root folder ID to be empty by default")
	}
}

func TestLoadGoogleDriveConfig(t *testing.T) {
	// Save original env vars
	originalKey := os.Getenv("GOOGLE_DRIVE_SERVICE_ACCOUNT_KEY")
	originalRoot := os.Getenv("GOOGLE_DRIVE_ROOT_FOLDER_ID") 
	originalApp := os.Getenv("GOOGLE_DRIVE_APPLICATION_NAME")
	
	// Clean up
	defer func() {
		os.Setenv("GOOGLE_DRIVE_SERVICE_ACCOUNT_KEY", originalKey)
		os.Setenv("GOOGLE_DRIVE_ROOT_FOLDER_ID", originalRoot)
		os.Setenv("GOOGLE_DRIVE_APPLICATION_NAME", originalApp)
	}()
	
	// Test with environment variables set
	os.Setenv("GOOGLE_DRIVE_SERVICE_ACCOUNT_KEY", "/path/to/key.json")
	os.Setenv("GOOGLE_DRIVE_ROOT_FOLDER_ID", "1234567890")
	os.Setenv("GOOGLE_DRIVE_APPLICATION_NAME", "Test App")
	
	config := loadGoogleDriveConfig()
	
	if config.ServiceAccountKey != "/path/to/key.json" {
		t.Errorf("Expected service account key '/path/to/key.json', got '%s'", config.ServiceAccountKey)
	}
	
	if config.RootFolderID != "1234567890" {
		t.Errorf("Expected root folder ID '1234567890', got '%s'", config.RootFolderID)
	}
	
	if config.ApplicationName != "Test App" {
		t.Errorf("Expected application name 'Test App', got '%s'", config.ApplicationName)
	}
}

func TestNewGoogleDriveFileSystem_RequiresServiceAccountKey(t *testing.T) {
	config := &GoogleDriveConfig{
		ApplicationName: "Test App",
	}
	
	_, err := NewGoogleDriveFileSystem(config)
	if err == nil {
		t.Error("Expected error when service account key is missing")
	}
	
	if err.Error() != "service account key file path is required" {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

// Integration test - only runs if GOOGLE_DRIVE_SERVICE_ACCOUNT_KEY is set
func TestGoogleDriveFileSystem_Integration(t *testing.T) {
	keyFile := os.Getenv("GOOGLE_DRIVE_SERVICE_ACCOUNT_KEY")
	if keyFile == "" {
		t.Skip("Skipping integration test - GOOGLE_DRIVE_SERVICE_ACCOUNT_KEY not set")
	}
	
	config := &GoogleDriveConfig{
		ServiceAccountKey: keyFile,
		ApplicationName:   "Curator Test",
	}
	
	gfs, err := NewGoogleDriveFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create Google Drive filesystem: %v", err)
	}
	
	// Test root folder access
	rootID := gfs.GetRootFolderID()
	if rootID == "" {
		t.Error("Expected non-empty root folder ID")
	}
	
	// Test listing root folder
	files, err := gfs.List("/")
	if err != nil {
		t.Fatalf("Failed to list root folder: %v", err)
	}
	
	t.Logf("Found %d files in root folder", len(files))
	
	// Test path resolution
	exists, err := gfs.Exists("/")
	if err != nil {
		t.Fatalf("Failed to check if root exists: %v", err)
	}
	if !exists {
		t.Error("Root folder should exist")
	}
	
	// Test non-existent path
	exists, err = gfs.Exists("/non-existent-folder-12345")
	if err != nil {
		t.Fatalf("Failed to check if non-existent path exists: %v", err)
	}
	if exists {
		t.Error("Non-existent path should not exist")
	}
}

func TestGoogleDriveFileSystem_CreateAndDeleteFolder(t *testing.T) {
	keyFile := os.Getenv("GOOGLE_DRIVE_SERVICE_ACCOUNT_KEY")
	if keyFile == "" {
		t.Skip("Skipping integration test - GOOGLE_DRIVE_SERVICE_ACCOUNT_KEY not set")
	}
	
	config := &GoogleDriveConfig{
		ServiceAccountKey: keyFile,
		ApplicationName:   "Curator Test",
	}
	
	gfs, err := NewGoogleDriveFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create Google Drive filesystem: %v", err)
	}
	
	testFolderName := "curator-test-folder-" + time.Now().Format("20060102-150405")
	testFolderPath := "/" + testFolderName
	
	// Create test folder
	err = gfs.CreateFolder(testFolderPath)
	if err != nil {
		t.Fatalf("Failed to create test folder: %v", err)
	}
	
	// Verify folder exists
	exists, err := gfs.Exists(testFolderPath)
	if err != nil {
		t.Fatalf("Failed to check if test folder exists: %v", err)
	}
	if !exists {
		t.Error("Test folder should exist after creation")
	}
	
	// List root to find our folder
	files, err := gfs.List("/")
	if err != nil {
		t.Fatalf("Failed to list root folder: %v", err)
	}
	
	found := false
	for _, file := range files {
		if file.Name() == testFolderName && file.IsDir() {
			found = true
			
			// Test file info
			if file.Path() != testFolderPath {
				t.Errorf("Expected path '%s', got '%s'", testFolderPath, file.Path())
			}
			
			if file.MimeType() != "application/vnd.google-apps.folder" {
				t.Errorf("Expected folder MIME type, got '%s'", file.MimeType())
			}
			
			if file.Hash() != "" {
				t.Error("Folders should not have a hash")
			}
			
			break
		}
	}
	
	if !found {
		t.Error("Created folder not found in listing")
	}
	
	// Clean up - delete test folder
	err = gfs.Delete(testFolderPath)
	if err != nil {
		t.Errorf("Failed to delete test folder: %v", err)
	}
}

func TestConfig_GoogleDriveIntegration(t *testing.T) {
	// Save original env vars
	originalFSType := os.Getenv("CURATOR_FILESYSTEM_TYPE")
	originalKey := os.Getenv("GOOGLE_DRIVE_SERVICE_ACCOUNT_KEY")
	
	// Clean up
	defer func() {
		os.Setenv("CURATOR_FILESYSTEM_TYPE", originalFSType)
		os.Setenv("GOOGLE_DRIVE_SERVICE_ACCOUNT_KEY", originalKey)
	}()
	
	// Test config loading with Google Drive
	os.Setenv("CURATOR_FILESYSTEM_TYPE", "googledrive")
	os.Setenv("GOOGLE_DRIVE_SERVICE_ACCOUNT_KEY", "/path/to/key.json")
	
	config := LoadConfig()
	
	if config.FileSystem.Type != "googledrive" {
		t.Errorf("Expected filesystem type 'googledrive', got '%s'", config.FileSystem.Type)
	}
	
	if config.FileSystem.GoogleDrive == nil {
		t.Error("Expected Google Drive config to be loaded")
	}
	
	if config.FileSystem.GoogleDrive.ServiceAccountKey != "/path/to/key.json" {
		t.Errorf("Expected service account key '/path/to/key.json', got '%s'", 
			config.FileSystem.GoogleDrive.ServiceAccountKey)
	}
}

func TestConfig_ValidateGoogleDrive(t *testing.T) {
	// Test valid Google Drive config
	config := &Config{
		AI: AIConfig{
			Provider: "mock",
		},
		FileSystem: FileSystemConfig{
			Type: "googledrive",
			GoogleDrive: &GoogleDriveConfig{
				ServiceAccountKey: "/path/to/key.json",
				ApplicationName:   "Test App",
			},
		},
	}
	
	err := config.Validate()
	if err != nil {
		t.Errorf("Valid Google Drive config should pass validation: %v", err)
	}
	
	// Test missing Google Drive config
	config.FileSystem.GoogleDrive = nil
	err = config.Validate()
	if err == nil {
		t.Error("Expected validation error when Google Drive config is missing")
	}
	
	// Test missing service account key
	config.FileSystem.GoogleDrive = &GoogleDriveConfig{
		ApplicationName: "Test App",
	}
	err = config.Validate()
	if err == nil {
		t.Error("Expected validation error when service account key is missing")
	}
}

func TestConfig_CreateGoogleDriveFileSystem(t *testing.T) {
	config := &Config{
		FileSystem: FileSystemConfig{
			Type: "googledrive",
			GoogleDrive: &GoogleDriveConfig{
				ServiceAccountKey: "/nonexistent/key.json",
				ApplicationName:   "Test App",
			},
		},
	}
	
	// This should fail because the key file doesn't exist
	_, err := config.CreateFileSystem()
	if err == nil {
		t.Error("Expected error when creating filesystem with non-existent key file")
	}
	
	// Test missing config
	config.FileSystem.GoogleDrive = nil
	_, err = config.CreateFileSystem()
	if err == nil {
		t.Error("Expected error when Google Drive config is missing")
	}
	
	expectedError := "Google Drive configuration is required when filesystem is 'googledrive'"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestGoogleDriveFileInfo(t *testing.T) {
	// Test Google Drive file info with mock data
	fileInfo := &googleDriveFileInfo{
		id:       "1234567890",
		name:     "test-file.txt",
		path:     "/test-file.txt",
		isDir:    false,
		size:     1024,
		modTime:  time.Now(),
		mimeType: "text/plain",
		hash:     "abcdef123456",
	}
	
	if fileInfo.Name() != "test-file.txt" {
		t.Errorf("Expected name 'test-file.txt', got '%s'", fileInfo.Name())
	}
	
	if fileInfo.Path() != "/test-file.txt" {
		t.Errorf("Expected path '/test-file.txt', got '%s'", fileInfo.Path())
	}
	
	if fileInfo.IsDir() {
		t.Error("Expected file to not be a directory")
	}
	
	if fileInfo.Size() != 1024 {
		t.Errorf("Expected size 1024, got %d", fileInfo.Size())
	}
	
	if fileInfo.MimeType() != "text/plain" {
		t.Errorf("Expected MIME type 'text/plain', got '%s'", fileInfo.MimeType())
	}
	
	if fileInfo.Hash() != "abcdef123456" {
		t.Errorf("Expected hash 'abcdef123456', got '%s'", fileInfo.Hash())
	}
	
	// Test directory file info
	dirInfo := &googleDriveFileInfo{
		id:       "9876543210",
		name:     "test-folder",
		path:     "/test-folder",
		isDir:    true,
		size:     0,
		modTime:  time.Now(),
		mimeType: "application/vnd.google-apps.folder",
	}
	
	if !dirInfo.IsDir() {
		t.Error("Expected directory to be a directory")
	}
	
	if dirInfo.Hash() != "" {
		t.Error("Expected empty hash for directory")
	}
	
	if dirInfo.MimeType() != "application/vnd.google-apps.folder" {
		t.Errorf("Expected folder MIME type, got '%s'", dirInfo.MimeType())
	}
}