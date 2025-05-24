package curator

import (
	"crypto/md5"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

// FileUtilities provides common file operations for different filesystem implementations
type FileUtilities struct{}

// NewFileUtilities creates a new FileUtilities instance
func NewFileUtilities() *FileUtilities {
	return &FileUtilities{}
}

// ComputeHashFromBytes computes MD5 hash from byte content
func (fu *FileUtilities) ComputeHashFromBytes(content []byte) string {
	hash := md5.Sum(content)
	return fmt.Sprintf("%x", hash)
}

// ComputeHashFromReader computes MD5 hash from an io.Reader
func (fu *FileUtilities) ComputeHashFromReader(reader io.Reader) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// ComputeHashFromFile computes MD5 hash from a file path
func (fu *FileUtilities) ComputeHashFromFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	return fu.ComputeHashFromReader(file)
}

// DetectMimeTypeFromExtension determines MIME type from file extension
func (fu *FileUtilities) DetectMimeTypeFromExtension(filename string) string {
	ext := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(ext)
	
	if mimeType != "" {
		return mimeType
	}
	
	// Return a default MIME type if extension is unknown
	return "application/octet-stream"
}

// DetectMimeTypeFromContent determines MIME type from file content
func (fu *FileUtilities) DetectMimeTypeFromContent(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	// Read first 512 bytes to detect content type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}
	
	// Use Go's content detection
	contentType := http.DetectContentType(buffer[:n])
	return contentType, nil
}

// DetectMimeType detects MIME type using both extension and content analysis
func (fu *FileUtilities) DetectMimeType(filePath string) string {
	// First try extension-based detection
	filename := filepath.Base(filePath)
	mimeType := fu.DetectMimeTypeFromExtension(filename)
	
	// If extension gave us a result, use it
	if mimeType != "application/octet-stream" {
		return mimeType
	}
	
	// Fall back to content-based detection
	if contentType, err := fu.DetectMimeTypeFromContent(filePath); err == nil {
		return contentType
	}
	
	// Default fallback
	return "application/octet-stream"
}

// DirectoryMimeType returns the standard MIME type for directories
func (fu *FileUtilities) DirectoryMimeType() string {
	return "inode/directory"
}

// CreateHash creates an MD5 hash for arbitrary data (used for composite hashes)
func (fu *FileUtilities) CreateHash(data string) string {
	hash := md5.New()
	hash.Write([]byte(data))
	return fmt.Sprintf("%x", hash.Sum(nil))
}