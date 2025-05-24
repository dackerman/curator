package curator

import (
	"crypto/md5"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MemoryFileSystem implements FileSystem interface for testing
type MemoryFileSystem struct {
	files map[string]*memoryFile
}

type memoryFile struct {
	name     string
	path     string
	isDir    bool
	size     int64
	modTime  time.Time
	content  []byte
	mimeType string
}

// NewMemoryFileSystem creates a new in-memory filesystem
func NewMemoryFileSystem() *MemoryFileSystem {
	return &MemoryFileSystem{
		files: make(map[string]*memoryFile),
	}
}

// AddFile adds a file to the memory filesystem
func (mfs *MemoryFileSystem) AddFile(path string, content []byte, mimeType string) {
	path = filepath.Clean(path)
	name := filepath.Base(path)
	
	mfs.files[path] = &memoryFile{
		name:     name,
		path:     path,
		isDir:    false,
		size:     int64(len(content)),
		modTime:  time.Now(),
		content:  content,
		mimeType: mimeType,
	}
	
	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		mfs.CreateFolder(dir)
	}
}

// AddFolder adds a directory to the memory filesystem
func (mfs *MemoryFileSystem) AddFolder(path string) {
	path = filepath.Clean(path)
	name := filepath.Base(path)
	
	if _, exists := mfs.files[path]; !exists {
		mfs.files[path] = &memoryFile{
			name:    name,
			path:    path,
			isDir:   true,
			size:    0,
			modTime: time.Now(),
		}
	}
	
	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		mfs.CreateFolder(dir)
	}
}

// List implements FileSystem.List
func (mfs *MemoryFileSystem) List(path string) ([]FileInfo, error) {
	path = filepath.Clean(path)
	
	var files []FileInfo
	for filePath, file := range mfs.files {
		dir := filepath.Dir(filePath)
		if dir == path || (path == "/" && !strings.Contains(strings.TrimPrefix(filePath, "/"), "/")) {
			files = append(files, &memoryFileInfo{file})
		}
	}
	
	// Sort by name for consistent ordering
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})
	
	return files, nil
}

// Read implements FileSystem.Read
func (mfs *MemoryFileSystem) Read(path string) (io.ReadCloser, error) {
	path = filepath.Clean(path)
	
	file, exists := mfs.files[path]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	
	if file.isDir {
		return nil, fmt.Errorf("cannot read directory: %s", path)
	}
	
	return io.NopCloser(strings.NewReader(string(file.content))), nil
}

// Move implements FileSystem.Move
func (mfs *MemoryFileSystem) Move(source, destination string) error {
	source = filepath.Clean(source)
	destination = filepath.Clean(destination)
	
	file, exists := mfs.files[source]
	if !exists {
		return fmt.Errorf("source file not found: %s", source)
	}
	
	// Check if destination already exists
	if _, exists := mfs.files[destination]; exists {
		return fmt.Errorf("destination already exists: %s", destination)
	}
	
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(destination)
	if dir != "." && dir != "/" {
		mfs.CreateFolder(dir)
	}
	
	// Create new file at destination
	newFile := &memoryFile{
		name:     filepath.Base(destination),
		path:     destination,
		isDir:    file.isDir,
		size:     file.size,
		modTime:  time.Now(),
		content:  file.content,
		mimeType: file.mimeType,
	}
	
	mfs.files[destination] = newFile
	delete(mfs.files, source)
	
	// If moving a directory, also move all its contents
	if file.isDir {
		// First collect all paths that need to be moved to avoid concurrent modification
		var pathsToMove []string
		for filePath := range mfs.files {
			if strings.HasPrefix(filePath, source+"/") {
				pathsToMove = append(pathsToMove, filePath)
			}
		}
		
		// Now move the collected paths
		for _, filePath := range pathsToMove {
			newPath := strings.Replace(filePath, source, destination, 1)
			mfs.files[newPath] = mfs.files[filePath]
			mfs.files[newPath].path = newPath
			delete(mfs.files, filePath)
		}
	}
	
	return nil
}

// CreateFolder implements FileSystem.CreateFolder
func (mfs *MemoryFileSystem) CreateFolder(path string) error {
	path = filepath.Clean(path)
	
	if _, exists := mfs.files[path]; exists {
		return nil // Already exists
	}
	
	mfs.AddFolder(path)
	return nil
}

// Delete implements FileSystem.Delete
func (mfs *MemoryFileSystem) Delete(path string) error {
	path = filepath.Clean(path)
	
	file, exists := mfs.files[path]
	if !exists {
		return fmt.Errorf("file not found: %s", path)
	}
	
	delete(mfs.files, path)
	
	// If deleting a directory, also delete all its contents
	if file.isDir {
		for filePath := range mfs.files {
			if strings.HasPrefix(filePath, path+"/") {
				delete(mfs.files, filePath)
			}
		}
	}
	
	return nil
}

// Exists implements FileSystem.Exists
func (mfs *MemoryFileSystem) Exists(path string) (bool, error) {
	path = filepath.Clean(path)
	_, exists := mfs.files[path]
	return exists, nil
}

// memoryFileInfo implements FileInfo interface
type memoryFileInfo struct {
	file *memoryFile
}

func (mfi *memoryFileInfo) Name() string {
	return mfi.file.name
}

func (mfi *memoryFileInfo) Path() string {
	return mfi.file.path
}

func (mfi *memoryFileInfo) IsDir() bool {
	return mfi.file.isDir
}

func (mfi *memoryFileInfo) Size() int64 {
	return mfi.file.size
}

func (mfi *memoryFileInfo) ModTime() time.Time {
	return mfi.file.modTime
}

func (mfi *memoryFileInfo) Hash() string {
	if mfi.file.isDir {
		return ""
	}
	hash := md5.Sum(mfi.file.content)
	return fmt.Sprintf("%x", hash)
}

func (mfi *memoryFileInfo) MimeType() string {
	return mfi.file.mimeType
}