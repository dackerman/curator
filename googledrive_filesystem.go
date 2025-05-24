package curator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GoogleDriveConfig holds configuration for Google Drive filesystem
type GoogleDriveConfig struct {
	// OAuth2CredentialsFile is the path to the OAuth2 client credentials JSON file
	OAuth2CredentialsFile string
	// OAuth2TokenFile is the path to store refresh/access tokens (optional)
	OAuth2TokenFile string
	// RootFolderID is the ID of the root folder to operate within (optional, defaults to entire Drive)
	RootFolderID string
	// ApplicationName is the name used to identify this application
	ApplicationName string
}

// DefaultGoogleDriveConfig returns default configuration for Google Drive
func DefaultGoogleDriveConfig() *GoogleDriveConfig {
	return &GoogleDriveConfig{
		ApplicationName: "Curator File Organizer",
	}
}

// GoogleDriveFileSystem implements FileSystem interface for Google Drive
type GoogleDriveFileSystem struct {
	config  *GoogleDriveConfig
	service *drive.Service
	rootID  string // Actual root folder ID to use
	utils   *FileUtilities
}

// OAuth2TokenInfo represents the stored OAuth2 tokens
type OAuth2TokenInfo struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

// OAuth2TokenManager handles OAuth2 token storage and refresh
type OAuth2TokenManager struct {
	credentialsFile string
	tokenFile       string
	config          *oauth2.Config
}

// NewOAuth2TokenManager creates a new OAuth2 token manager
func NewOAuth2TokenManager(credentialsFile, tokenFile string) (*OAuth2TokenManager, error) {
	if credentialsFile == "" {
		return nil, fmt.Errorf("OAuth2 credentials file path is required")
	}

	// Read OAuth2 credentials
	credData, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read OAuth2 credentials file: %w", err)
	}

	// Parse OAuth2 config from credentials file
	config, err := google.ConfigFromJSON(credData, drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OAuth2 credentials: %w", err)
	}

	// Set redirect URI for local server
	config.RedirectURL = "http://localhost:8080/oauth2callback"

	return &OAuth2TokenManager{
		credentialsFile: credentialsFile,
		tokenFile:       tokenFile,
		config:          config,
	}, nil
}

// GetValidToken returns a valid OAuth2 token, refreshing if necessary
func (tm *OAuth2TokenManager) GetValidToken(ctx context.Context) (*oauth2.Token, error) {
	// Try to load existing token
	token, err := tm.loadToken()
	if err != nil {
		// No existing token, need to perform initial OAuth2 flow
		return tm.performOAuth2Flow(ctx)
	}

	// Check if token needs refresh
	if time.Now().Add(5 * time.Minute).After(token.Expiry) {
		// Token is close to expiry, refresh it
		tokenSource := tm.config.TokenSource(ctx, token)
		newToken, err := tokenSource.Token()
		if err != nil {
			// Refresh failed, perform new OAuth2 flow
			return tm.performOAuth2Flow(ctx)
		}
		
		// Save refreshed token
		if err := tm.saveToken(newToken); err != nil {
			return nil, fmt.Errorf("failed to save refreshed token: %w", err)
		}
		
		return newToken, nil
	}

	return token, nil
}

// loadToken loads OAuth2 token from file
func (tm *OAuth2TokenManager) loadToken() (*oauth2.Token, error) {
	if tm.tokenFile == "" {
		return nil, fmt.Errorf("no token file specified")
	}

	data, err := os.ReadFile(tm.tokenFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var tokenInfo OAuth2TokenInfo
	if err := json.Unmarshal(data, &tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &oauth2.Token{
		AccessToken:  tokenInfo.AccessToken,
		RefreshToken: tokenInfo.RefreshToken,
		TokenType:    tokenInfo.TokenType,
		Expiry:       tokenInfo.Expiry,
	}, nil
}

// saveToken saves OAuth2 token to file
func (tm *OAuth2TokenManager) saveToken(token *oauth2.Token) error {
	if tm.tokenFile == "" {
		return fmt.Errorf("no token file specified")
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(tm.tokenFile), 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	tokenInfo := OAuth2TokenInfo{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}

	data, err := json.MarshalIndent(tokenInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(tm.tokenFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// performOAuth2Flow performs the complete OAuth2 authorization flow
func (tm *OAuth2TokenManager) performOAuth2Flow(ctx context.Context) (*oauth2.Token, error) {
	// Start local HTTP server for OAuth2 callback
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	
	server := &http.Server{Addr: ":8080"}
	http.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no authorization code received")
			return
		}
		
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html>
				<body>
					<h2>Authorization successful!</h2>
					<p>You can close this window and return to Curator.</p>
				</body>
			</html>
		`))
		
		codeCh <- code
	})

	// Start server in background
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- fmt.Errorf("failed to start OAuth2 callback server: %w", err)
		}
	}()

	// Generate authorization URL and open browser
	authURL := tm.config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	fmt.Printf("\n🔐 Opening browser for Google Drive authorization...\n")
	fmt.Printf("If browser doesn't open automatically, visit:\n%s\n\n", authURL)
	
	// Try to open browser automatically
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser automatically: %v\n", err)
	}

	// Wait for authorization code or error
	var code string
	select {
	case code = <-codeCh:
		// Got authorization code
	case err := <-errCh:
		server.Shutdown(ctx)
		return nil, err
	case <-ctx.Done():
		server.Shutdown(ctx)
		return nil, fmt.Errorf("OAuth2 flow cancelled")
	}

	// Shutdown callback server
	server.Shutdown(ctx)

	// Exchange authorization code for token
	token, err := tm.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange authorization code for token: %w", err)
	}

	// Save token for future use
	if err := tm.saveToken(token); err != nil {
		return nil, fmt.Errorf("failed to save OAuth2 token: %w", err)
	}

	fmt.Printf("✅ Google Drive authorization successful!\n\n")
	return token, nil
}

// openBrowser attempts to open the specified URL in the user's default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch {
	case os.Getenv("WSL_DISTRO_NAME") != "":
		// Windows Subsystem for Linux
		cmd = "cmd.exe"
		args = []string{"/c", "start", url}
	case fileExists("/usr/bin/xdg-open"):
		// Linux with xdg-open
		cmd = "xdg-open"
		args = []string{url}
	case fileExists("/usr/bin/open"):
		// macOS
		cmd = "open"
		args = []string{url}
	default:
		return fmt.Errorf("unable to detect browser command for this platform")
	}

	return execCommand(cmd, args...)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// execCommand executes a command with arguments
func execCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Start()
}

// NewGoogleDriveFileSystem creates a new Google Drive filesystem instance
func NewGoogleDriveFileSystem(config *GoogleDriveConfig) (*GoogleDriveFileSystem, error) {
	if config.OAuth2CredentialsFile == "" {
		return nil, fmt.Errorf("OAuth2 credentials file path is required (set GOOGLE_DRIVE_OAUTH_CREDENTIALS)")
	}

	ctx := context.Background()

	// Set default token file if not specified
	tokenFile := config.OAuth2TokenFile
	if tokenFile == "" {
		// Use default location in curator store directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			tokenFile = ".curator/google_tokens.json"
		} else {
			tokenFile = filepath.Join(homeDir, ".curator", "google_tokens.json")
		}
	}

	// Create OAuth2 token manager
	tokenManager, err := NewOAuth2TokenManager(config.OAuth2CredentialsFile, tokenFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth2 token manager: %w", err)
	}

	// Get valid OAuth2 token (will perform browser flow if needed)
	token, err := tokenManager.GetValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth2 token: %w", err)
	}

	// Create Drive service with OAuth2 token
	service, err := drive.NewService(ctx, option.WithTokenSource(
		tokenManager.config.TokenSource(ctx, token),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive service: %w", err)
	}

	// Determine root folder ID  
	rootID := config.RootFolderID
	if rootID == "" {
		// Use user's root folder (entire Drive)
		rootID = "root"
	}

	// Verify we can access the root folder
	_, err = service.Files.Get(rootID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to access root folder %s: %w", rootID, err)
	}

	return &GoogleDriveFileSystem{
		config:  config,
		service: service,
		rootID:  rootID,
		utils:   NewFileUtilities(),
	}, nil
}

// GetRootFolderID returns the root folder ID
func (gfs *GoogleDriveFileSystem) GetRootFolderID() string {
	return gfs.rootID
}

// resolvePath converts a path to a Google Drive file ID
// Paths are in format: /folder1/folder2/file.txt
func (gfs *GoogleDriveFileSystem) resolvePath(path string) (string, error) {
	// Clean and normalize the path
	path = filepath.Clean(path)
	path = strings.TrimPrefix(path, "/")
	
	// If empty path, return root
	if path == "" || path == "." {
		return gfs.rootID, nil
	}
	
	// Split path into components
	parts := strings.Split(path, "/")
	currentID := gfs.rootID
	
	// Traverse each path component
	for _, part := range parts {
		if part == "" {
			continue
		}
		
		// Find child with this name in current folder
		query := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false", 
			strings.ReplaceAll(part, "'", "\\'"), currentID)
		
		fileList, err := gfs.service.Files.List().Q(query).Fields("files(id, name, mimeType)").Do()
		if err != nil {
			return "", fmt.Errorf("failed to search for %s: %w", part, err)
		}
		
		if len(fileList.Files) == 0 {
			return "", fmt.Errorf("path not found: %s", path)
		}
		
		if len(fileList.Files) > 1 {
			return "", fmt.Errorf("ambiguous path - multiple files named %s", part)
		}
		
		currentID = fileList.Files[0].Id
	}
	
	return currentID, nil
}

// pathToID is a helper that returns both the file ID and any error
func (gfs *GoogleDriveFileSystem) pathToID(path string) (string, error) {
	return gfs.resolvePath(path)
}

// List implements FileSystem.List
func (gfs *GoogleDriveFileSystem) List(path string) ([]FileInfo, error) {
	folderID, err := gfs.pathToID(path)
	if err != nil {
		return nil, err
	}
	
	// Verify it's a folder
	file, err := gfs.service.Files.Get(folderID).Fields("mimeType").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get folder info: %w", err)
	}
	
	if file.MimeType != "application/vnd.google-apps.folder" {
		return nil, fmt.Errorf("path is not a folder: %s", path)
	}
	
	// List files in folder
	query := fmt.Sprintf("'%s' in parents and trashed=false", folderID)
	fileList, err := gfs.service.Files.List().
		Q(query).
		Fields("files(id, name, mimeType, size, modifiedTime, md5Checksum, parents)").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	
	var files []FileInfo
	for _, file := range fileList.Files {
		// Build the full path for this file
		filePath := filepath.Join(path, file.Name)
		if !strings.HasPrefix(filePath, "/") {
			filePath = "/" + filePath
		}
		filePath = filepath.ToSlash(filePath)
		
		fileInfo := &googleDriveFileInfo{
			id:       file.Id,
			name:     file.Name,
			path:     filePath,
			isDir:    file.MimeType == "application/vnd.google-apps.folder",
			size:     file.Size,
			mimeType: file.MimeType,
			hash:     file.Md5Checksum,
			service:  gfs.service,
		}
		
		// Parse modification time
		if file.ModifiedTime != "" {
			if modTime, err := time.Parse(time.RFC3339, file.ModifiedTime); err == nil {
				fileInfo.modTime = modTime
			}
		}
		
		files = append(files, fileInfo)
	}
	
	return files, nil
}

// Read implements FileSystem.Read
func (gfs *GoogleDriveFileSystem) Read(path string) (io.ReadCloser, error) {
	fileID, err := gfs.pathToID(path)
	if err != nil {
		return nil, err
	}
	
	// Verify it's not a folder
	file, err := gfs.service.Files.Get(fileID).Fields("mimeType").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	
	if file.MimeType == "application/vnd.google-apps.folder" {
		return nil, fmt.Errorf("cannot read directory: %s", path)
	}
	
	// Download file content
	resp, err := gfs.service.Files.Get(fileID).Download()
	if err != nil {
		return nil, fmt.Errorf("failed to download file %s: %w", path, err)
	}
	
	return resp.Body, nil
}

// Move implements FileSystem.Move
func (gfs *GoogleDriveFileSystem) Move(source, destination string) error {
	sourceID, err := gfs.pathToID(source)
	if err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	
	// Parse destination path
	destDir := filepath.Dir(destination)
	destName := filepath.Base(destination)
	
	destDirID, err := gfs.pathToID(destDir)
	if err != nil {
		return fmt.Errorf("invalid destination directory: %w", err)
	}
	
	// Get current file info to get current parents
	file, err := gfs.service.Files.Get(sourceID).Fields("parents").Do()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}
	
	// Update file: change name and parent
	update := &drive.File{
		Name: destName,
	}
	
	// Move by removing old parents and adding new parent
	var removeParents []string
	for _, parent := range file.Parents {
		removeParents = append(removeParents, parent)
	}
	
	_, err = gfs.service.Files.Update(sourceID, update).
		AddParents(destDirID).
		RemoveParents(strings.Join(removeParents, ",")).
		Do()
	if err != nil {
		return fmt.Errorf("failed to move %s to %s: %w", source, destination, err)
	}
	
	return nil
}

// CreateFolder implements FileSystem.CreateFolder
func (gfs *GoogleDriveFileSystem) CreateFolder(path string) error {
	// Parse path
	parentDir := filepath.Dir(path)
	folderName := filepath.Base(path)
	
	parentID, err := gfs.pathToID(parentDir)
	if err != nil {
		return fmt.Errorf("invalid parent directory: %w", err)
	}
	
	// Check if folder already exists
	query := fmt.Sprintf("name='%s' and '%s' in parents and mimeType='application/vnd.google-apps.folder' and trashed=false", 
		strings.ReplaceAll(folderName, "'", "\\'"), parentID)
	
	fileList, err := gfs.service.Files.List().Q(query).Do()
	if err != nil {
		return fmt.Errorf("failed to check if folder exists: %w", err)
	}
	
	if len(fileList.Files) > 0 {
		return fmt.Errorf("folder already exists: %s", path)
	}
	
	// Create folder
	folder := &drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{parentID},
	}
	
	_, err = gfs.service.Files.Create(folder).Do()
	if err != nil {
		return fmt.Errorf("failed to create folder %s: %w", path, err)
	}
	
	return nil
}

// Delete implements FileSystem.Delete
func (gfs *GoogleDriveFileSystem) Delete(path string) error {
	fileID, err := gfs.pathToID(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	
	// Move to trash instead of permanent delete for safety
	_, err = gfs.service.Files.Update(fileID, &drive.File{Trashed: true}).Do()
	if err != nil {
		return fmt.Errorf("failed to delete %s: %w", path, err)
	}
	
	return nil
}

// Exists implements FileSystem.Exists
func (gfs *GoogleDriveFileSystem) Exists(path string) (bool, error) {
	_, err := gfs.pathToID(path)
	if err != nil {
		if strings.Contains(err.Error(), "path not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// googleDriveFileInfo implements FileInfo interface for Google Drive files
type googleDriveFileInfo struct {
	id       string
	name     string
	path     string
	isDir    bool
	size     int64
	modTime  time.Time
	mimeType string
	hash     string
	service  *drive.Service
}

func (gdfi *googleDriveFileInfo) Name() string {
	return gdfi.name
}

func (gdfi *googleDriveFileInfo) Path() string {
	return gdfi.path
}

func (gdfi *googleDriveFileInfo) IsDir() bool {
	return gdfi.isDir
}

func (gdfi *googleDriveFileInfo) Size() int64 {
	return gdfi.size
}

func (gdfi *googleDriveFileInfo) ModTime() time.Time {
	return gdfi.modTime
}

func (gdfi *googleDriveFileInfo) Hash() string {
	if gdfi.isDir {
		return ""
	}
	
	// Use MD5 checksum from Drive API if available
	if gdfi.hash != "" {
		return gdfi.hash
	}
	
	// For Google Workspace files, we can't get MD5, so compute a simple hash
	// based on file ID and modification time using shared utilities
	if strings.HasPrefix(gdfi.mimeType, "application/vnd.google-apps.") {
		utils := NewFileUtilities()
		return utils.CreateHash(gdfi.id + gdfi.modTime.String())
	}
	
	return ""
}

func (gdfi *googleDriveFileInfo) MimeType() string {
	if gdfi.isDir {
		return "application/vnd.google-apps.folder"
	}
	return gdfi.mimeType
}