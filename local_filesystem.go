package curator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalFileSystem implements FileSystem interface for the local filesystem
type LocalFileSystem struct {
	rootPath string
	utils    *FileUtilities
}

// NewLocalFileSystem creates a new local filesystem instance
func NewLocalFileSystem(rootPath string) (*LocalFileSystem, error) {
	// Clean and validate the root path
	rootPath = filepath.Clean(rootPath)
	
	// Check if root path exists and is a directory
	info, err := os.Stat(rootPath)
	if err != nil {
		return nil, fmt.Errorf("root path does not exist: %w", err)
	}
	
	if !info.IsDir() {
		return nil, fmt.Errorf("root path is not a directory: %s", rootPath)
	}
	
	return &LocalFileSystem{
		rootPath: rootPath,
		utils:    NewFileUtilities(),
	}, nil
}

// GetRootPath returns the root path of the filesystem
func (lfs *LocalFileSystem) GetRootPath() string {
	return lfs.rootPath
}

// resolvePath converts a path to an absolute path within the root
func (lfs *LocalFileSystem) resolvePath(path string) (string, error) {
	// Clean the path first
	path = filepath.Clean(path)
	
	// Always strip leading slash to make it relative to root
	path = strings.TrimPrefix(path, "/")
	
	// Join with root path
	absPath := filepath.Join(lfs.rootPath, path)
	
	// Clean the result to normalize path separators and remove redundant elements
	absPath = filepath.Clean(absPath)
	
	// Ensure the resulting path is within the root directory
	rel, err := filepath.Rel(lfs.rootPath, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path is outside root directory: %s", path)
	}
	
	return absPath, nil
}

// List implements FileSystem.List
func (lfs *LocalFileSystem) List(path string) ([]FileInfo, error) {
	absPath, err := lfs.resolvePath(path)
	if err != nil {
		return nil, err
	}
	
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
	}
	
	var files []FileInfo
	for _, entry := range entries {
		entryPath := filepath.Join(absPath, entry.Name())
		
		// Get file info
		info, err := entry.Info()
		if err != nil {
			// Skip files we can't read
			continue
		}
		
		// Convert back to relative path
		relPath, err := filepath.Rel(lfs.rootPath, entryPath)
		if err != nil {
			continue
		}
		
		// Normalize path separators for consistency
		relPath = filepath.ToSlash(relPath)
		if !strings.HasPrefix(relPath, "/") {
			relPath = "/" + relPath
		}
		
		fileInfo := &localFileInfo{
			name:    entry.Name(),
			path:    relPath,
			isDir:   entry.IsDir(),
			size:    info.Size(),
			modTime: info.ModTime(),
			absPath: entryPath,
		}
		
		files = append(files, fileInfo)
	}
	
	return files, nil
}

// Read implements FileSystem.Read
func (lfs *LocalFileSystem) Read(path string) (io.ReadCloser, error) {
	absPath, err := lfs.resolvePath(path)
	if err != nil {
		return nil, err
	}
	
	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	
	return file, nil
}

// Move implements FileSystem.Move
func (lfs *LocalFileSystem) Move(source, destination string) error {
	srcPath, err := lfs.resolvePath(source)
	if err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	
	dstPath, err := lfs.resolvePath(destination)
	if err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}
	
	// Check if source exists
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("source does not exist: %w", err)
	}
	
	// Check if destination already exists
	if _, err := os.Stat(dstPath); err == nil {
		return fmt.Errorf("destination already exists: %s", destination)
	}
	
	// Create destination directory if it doesn't exist
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	// Perform the move
	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to move %s to %s: %w", source, destination, err)
	}
	
	return nil
}

// CreateFolder implements FileSystem.CreateFolder
func (lfs *LocalFileSystem) CreateFolder(path string) error {
	absPath, err := lfs.resolvePath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	
	// Create directory with proper permissions
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return fmt.Errorf("failed to create folder %s: %w", path, err)
	}
	
	return nil
}

// Delete implements FileSystem.Delete
func (lfs *LocalFileSystem) Delete(path string) error {
	absPath, err := lfs.resolvePath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	
	// Check if path exists
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("path does not exist: %w", err)
	}
	
	// Remove the file or directory
	if err := os.RemoveAll(absPath); err != nil {
		return fmt.Errorf("failed to delete %s: %w", path, err)
	}
	
	return nil
}

// Exists implements FileSystem.Exists
func (lfs *LocalFileSystem) Exists(path string) (bool, error) {
	absPath, err := lfs.resolvePath(path)
	if err != nil {
		return false, err
	}
	
	_, err = os.Stat(absPath)
	if err == nil {
		return true, nil
	}
	
	if os.IsNotExist(err) {
		return false, nil
	}
	
	return false, err
}

// localFileInfo implements FileInfo interface for local files
type localFileInfo struct {
	name    string
	path    string
	isDir   bool
	size    int64
	modTime time.Time
	absPath string
}

func (lfi *localFileInfo) Name() string {
	return lfi.name
}

func (lfi *localFileInfo) Path() string {
	return lfi.path
}

func (lfi *localFileInfo) IsDir() bool {
	return lfi.isDir
}

func (lfi *localFileInfo) Size() int64 {
	return lfi.size
}

func (lfi *localFileInfo) ModTime() time.Time {
	return lfi.modTime
}

func (lfi *localFileInfo) Hash() string {
	if lfi.isDir {
		return ""
	}
	
	// Compute MD5 hash of file contents using shared utilities
	utils := NewFileUtilities()
	hash, err := utils.ComputeHashFromFile(lfi.absPath)
	if err != nil {
		return ""
	}
	
	return hash
}

func (lfi *localFileInfo) MimeType() string {
	if lfi.isDir {
		utils := NewFileUtilities()
		return utils.DirectoryMimeType()
	}
	
	// Use shared utilities for MIME type detection
	utils := NewFileUtilities()
	return utils.DetectMimeType(lfi.absPath)
}