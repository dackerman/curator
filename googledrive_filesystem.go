package curator

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GoogleDriveConfig holds configuration for Google Drive filesystem
type GoogleDriveConfig struct {
	// ServiceAccountKey is the path to the service account JSON key file
	ServiceAccountKey string
	// RootFolderID is the ID of the root folder to operate within (optional)
	// If empty, uses the service account's root folder
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
}

// NewGoogleDriveFileSystem creates a new Google Drive filesystem instance
func NewGoogleDriveFileSystem(config *GoogleDriveConfig) (*GoogleDriveFileSystem, error) {
	if config.ServiceAccountKey == "" {
		return nil, fmt.Errorf("service account key file path is required")
	}

	ctx := context.Background()
	
	// Create Drive service with service account authentication
	service, err := drive.NewService(ctx, option.WithCredentialsFile(config.ServiceAccountKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive service: %w", err)
	}

	// Determine root folder ID
	rootID := config.RootFolderID
	if rootID == "" {
		// Use service account's root folder
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
	// based on file ID and modification time
	if strings.HasPrefix(gdfi.mimeType, "application/vnd.google-apps.") {
		hash := md5.New()
		hash.Write([]byte(gdfi.id + gdfi.modTime.String()))
		return fmt.Sprintf("%x", hash.Sum(nil))
	}
	
	return ""
}

func (gdfi *googleDriveFileInfo) MimeType() string {
	if gdfi.isDir {
		return "application/vnd.google-apps.folder"
	}
	return gdfi.mimeType
}