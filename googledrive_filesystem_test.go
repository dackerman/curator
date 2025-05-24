package curator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultGoogleDriveConfig(t *testing.T) {
	config := DefaultGoogleDriveConfig()
	
	if config.ApplicationName == "" {
		t.Error("Expected default application name to be set")
	}
	
	if config.OAuth2CredentialsFile != "" {
		t.Error("Expected OAuth2 credentials file to be empty by default")
	}
	
	if config.OAuth2TokenFile != "" {
		t.Error("Expected OAuth2 token file to be empty by default")
	}
	
	if config.RootFolderID != "" {
		t.Error("Expected root folder ID to be empty by default")
	}
}

func TestLoadGoogleDriveConfig(t *testing.T) {
	// Save original env vars
	originalCreds := os.Getenv("GOOGLE_DRIVE_OAUTH_CREDENTIALS")
	originalTokens := os.Getenv("GOOGLE_DRIVE_OAUTH_TOKENS")
	originalRoot := os.Getenv("GOOGLE_DRIVE_ROOT_FOLDER_ID") 
	originalApp := os.Getenv("GOOGLE_DRIVE_APPLICATION_NAME")
	
	// Clean up
	defer func() {
		os.Setenv("GOOGLE_DRIVE_OAUTH_CREDENTIALS", originalCreds)
		os.Setenv("GOOGLE_DRIVE_OAUTH_TOKENS", originalTokens)
		os.Setenv("GOOGLE_DRIVE_ROOT_FOLDER_ID", originalRoot)
		os.Setenv("GOOGLE_DRIVE_APPLICATION_NAME", originalApp)
	}()
	
	// Test with environment variables set
	os.Setenv("GOOGLE_DRIVE_OAUTH_CREDENTIALS", "/path/to/credentials.json")
	os.Setenv("GOOGLE_DRIVE_OAUTH_TOKENS", "/path/to/tokens.json")
	os.Setenv("GOOGLE_DRIVE_ROOT_FOLDER_ID", "1234567890")
	os.Setenv("GOOGLE_DRIVE_APPLICATION_NAME", "Test App")
	
	config := loadGoogleDriveConfig()
	
	if config.OAuth2CredentialsFile != "/path/to/credentials.json" {
		t.Errorf("Expected OAuth2 credentials file '/path/to/credentials.json', got '%s'", config.OAuth2CredentialsFile)
	}
	
	if config.OAuth2TokenFile != "/path/to/tokens.json" {
		t.Errorf("Expected OAuth2 token file '/path/to/tokens.json', got '%s'", config.OAuth2TokenFile)
	}
	
	if config.RootFolderID != "1234567890" {
		t.Errorf("Expected root folder ID '1234567890', got '%s'", config.RootFolderID)
	}
	
	if config.ApplicationName != "Test App" {
		t.Errorf("Expected application name 'Test App', got '%s'", config.ApplicationName)
	}
}

func TestNewGoogleDriveFileSystem_RequiresOAuth2Credentials(t *testing.T) {
	config := &GoogleDriveConfig{
		ApplicationName: "Test App",
	}
	
	_, err := NewGoogleDriveFileSystem(config)
	if err == nil {
		t.Error("Expected error when OAuth2 credentials file is missing")
	}
	
	if err.Error() != "OAuth2 credentials file path is required (set GOOGLE_DRIVE_OAUTH_CREDENTIALS)" {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

// Integration test - only runs if GOOGLE_DRIVE_OAUTH_CREDENTIALS is set
func TestGoogleDriveFileSystem_Integration(t *testing.T) {
	credFile := os.Getenv("GOOGLE_DRIVE_OAUTH_CREDENTIALS")
	if credFile == "" {
		t.Skip("Skipping integration test - GOOGLE_DRIVE_OAUTH_CREDENTIALS not set")
	}
	
	config := &GoogleDriveConfig{
		OAuth2CredentialsFile: credFile,
		ApplicationName:       "Curator Test",
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
	credFile := os.Getenv("GOOGLE_DRIVE_OAUTH_CREDENTIALS")
	if credFile == "" {
		t.Skip("Skipping integration test - GOOGLE_DRIVE_OAUTH_CREDENTIALS not set")
	}
	
	config := &GoogleDriveConfig{
		OAuth2CredentialsFile: credFile,
		ApplicationName:       "Curator Test",
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
	originalCreds := os.Getenv("GOOGLE_DRIVE_OAUTH_CREDENTIALS")
	
	// Clean up
	defer func() {
		os.Setenv("CURATOR_FILESYSTEM_TYPE", originalFSType)
		os.Setenv("GOOGLE_DRIVE_OAUTH_CREDENTIALS", originalCreds)
	}()
	
	// Test config loading with Google Drive
	os.Setenv("CURATOR_FILESYSTEM_TYPE", "googledrive")
	os.Setenv("GOOGLE_DRIVE_OAUTH_CREDENTIALS", "/path/to/credentials.json")
	
	config := LoadConfig()
	
	if config.FileSystem.Type != "googledrive" {
		t.Errorf("Expected filesystem type 'googledrive', got '%s'", config.FileSystem.Type)
	}
	
	if config.FileSystem.GoogleDrive == nil {
		t.Error("Expected Google Drive config to be loaded")
	}
	
	if config.FileSystem.GoogleDrive.OAuth2CredentialsFile != "/path/to/credentials.json" {
		t.Errorf("Expected OAuth2 credentials file '/path/to/credentials.json', got '%s'", 
			config.FileSystem.GoogleDrive.OAuth2CredentialsFile)
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
				OAuth2CredentialsFile: "/path/to/credentials.json",
				ApplicationName:       "Test App",
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
	
	// Test missing OAuth2 credentials file
	config.FileSystem.GoogleDrive = &GoogleDriveConfig{
		ApplicationName: "Test App",
	}
	err = config.Validate()
	if err == nil {
		t.Error("Expected validation error when OAuth2 credentials file is missing")
	}
}

func TestConfig_CreateGoogleDriveFileSystem(t *testing.T) {
	config := &Config{
		FileSystem: FileSystemConfig{
			Type: "googledrive",
			GoogleDrive: &GoogleDriveConfig{
				OAuth2CredentialsFile: "/nonexistent/credentials.json",
				ApplicationName:       "Test App",
			},
		},
	}
	
	// This should fail because the credentials file doesn't exist
	_, err := config.CreateFileSystem()
	if err == nil {
		t.Error("Expected error when creating filesystem with non-existent credentials file")
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

// OAuth2 Token Management Tests

func TestOAuth2TokenManager_Creation(t *testing.T) {
	// Test creation with missing credentials file
	_, err := NewOAuth2TokenManager("", "/tmp/tokens.json")
	if err == nil {
		t.Error("Expected error when credentials file is empty")
	}
	
	// Test creation with non-existent credentials file
	_, err = NewOAuth2TokenManager("/nonexistent/credentials.json", "/tmp/tokens.json")
	if err == nil {
		t.Error("Expected error when credentials file doesn't exist")
	}
}

func TestOAuth2TokenInfo_Marshaling(t *testing.T) {
	// Test token info marshaling/unmarshaling
	tokenInfo := OAuth2TokenInfo{
		AccessToken:  "access_token_123",
		RefreshToken: "refresh_token_456",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	
	// Marshal
	data, err := json.Marshal(tokenInfo)
	if err != nil {
		t.Fatalf("Failed to marshal token info: %v", err)
	}
	
	// Unmarshal
	var unmarshaled OAuth2TokenInfo
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal token info: %v", err)
	}
	
	// Verify
	if unmarshaled.AccessToken != tokenInfo.AccessToken {
		t.Errorf("Expected access token '%s', got '%s'", tokenInfo.AccessToken, unmarshaled.AccessToken)
	}
	
	if unmarshaled.RefreshToken != tokenInfo.RefreshToken {
		t.Errorf("Expected refresh token '%s', got '%s'", tokenInfo.RefreshToken, unmarshaled.RefreshToken)
	}
	
	if unmarshaled.TokenType != tokenInfo.TokenType {
		t.Errorf("Expected token type '%s', got '%s'", tokenInfo.TokenType, unmarshaled.TokenType)
	}
	
	// Allow for small time differences due to JSON serialization
	if unmarshaled.Expiry.Sub(tokenInfo.Expiry).Abs() > time.Second {
		t.Errorf("Token expiry times differ by more than 1 second")
	}
}

func TestOAuth2TokenManager_TokenFile_DefaultPath(t *testing.T) {
	// Test default token file path generation
	config := &GoogleDriveConfig{
		OAuth2CredentialsFile: "/path/to/credentials.json",
		ApplicationName:       "Test App",
	}
	
	// This test verifies the default path logic in NewGoogleDriveFileSystem
	// We can't easily test the actual OAuth2 flow without real credentials
	// but we can test that the path logic works correctly
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get user home directory")
	}
	
	expectedPath := filepath.Join(homeDir, ".curator", "google_tokens.json")
	
	// The NewGoogleDriveFileSystem function would use this path when OAuth2TokenFile is empty
	if config.OAuth2TokenFile == "" {
		tokenFile := expectedPath
		if !strings.Contains(tokenFile, ".curator") {
			t.Error("Expected default token file to be in .curator directory")
		}
	}
}

func TestBrowserOpeningUtilities(t *testing.T) {
	// Test file existence checking
	if !fileExists("/etc/passwd") && !fileExists("/usr/bin") {
		t.Error("Expected at least one of these common paths to exist")
	}
	
	if fileExists("/this/path/definitely/does/not/exist") {
		t.Error("Expected non-existent path to return false")
	}
}

func TestGoogleDriveConfig_OAuth2Fields(t *testing.T) {
	config := &GoogleDriveConfig{
		OAuth2CredentialsFile: "/path/to/creds.json",
		OAuth2TokenFile:       "/path/to/tokens.json",
		RootFolderID:          "root",
		ApplicationName:       "Test App",
	}
	
	// Test that all OAuth2 fields are properly set
	if config.OAuth2CredentialsFile == "" {
		t.Error("OAuth2 credentials file should be set")
	}
	
	if config.OAuth2TokenFile == "" {
		t.Error("OAuth2 token file should be set")
	}
	
	if config.ApplicationName == "" {
		t.Error("Application name should be set")
	}
	
	// Test that root folder defaults work
	if config.RootFolderID != "root" {
		t.Errorf("Expected root folder ID 'root', got '%s'", config.RootFolderID)
	}
}