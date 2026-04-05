package lsp

import (
	"fmt"
	"strings"
	"sync"
)

// FileID represents a unique identifier for a file within the package store
type FileID string

// PackageID represents a unique identifier for a package
type PackageID string

// FileInfo contains metadata about a file
type FileInfo struct {
	ID       FileID    `json:"id"`
	Path     string    `json:"path"`
	Filename string    `json:"filename"`
	Package  PackageID `json:"package"`
	Size     int64     `json:"size,omitempty"`
	Modified int64     `json:"modified,omitempty"`
}

// PackageInfo contains metadata about a package
type PackageInfo struct {
	ID    PackageID            `json:"id"`
	Name  string               `json:"name"`
	Path  string               `json:"path"`
	Files map[FileID]*FileInfo `json:"files"`
}

// PackageStore maintains a centralized store of packages and files
type PackageStore struct {
	mu       sync.RWMutex
	packages map[PackageID]*PackageInfo
	files    map[FileID]*FileInfo
	nextID   int
}

// NewPackageStore creates a new package store
func NewPackageStore() *PackageStore {
	return &PackageStore{
		packages: make(map[PackageID]*PackageInfo),
		files:    make(map[FileID]*FileInfo),
		nextID:   1,
	}
}

// RegisterFile registers a file in the package store and returns its FileID
func (ps *PackageStore) RegisterFile(filePath string, packagePath string) FileID {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Check if file already exists
	for id, file := range ps.files {
		if file.Path == filePath {
			return id
		}
	}

	// Generate new file ID
	fileID := FileID(fmt.Sprintf("f%d", ps.nextID))
	ps.nextID++

	// Extract filename
	filename := extractFilenameFromPath(filePath)

	// Create file info
	fileInfo := &FileInfo{
		ID:       fileID,
		Path:     filePath,
		Filename: filename,
		Package:  PackageID(packagePath),
	}

	// Register file
	ps.files[fileID] = fileInfo

	// Ensure package exists
	packageID := PackageID(packagePath)
	if _, exists := ps.packages[packageID]; !exists {
		ps.packages[packageID] = &PackageInfo{
			ID:    packageID,
			Name:  extractPackageName(packagePath),
			Path:  packagePath,
			Files: make(map[FileID]*FileInfo),
		}
	}

	// Add file to package
	ps.packages[packageID].Files[fileID] = fileInfo

	return fileID
}

// GetFileInfo returns file information by ID
func (ps *PackageStore) GetFileInfo(fileID FileID) (*FileInfo, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	file, exists := ps.files[fileID]
	return file, exists
}

// GetPackageInfo returns package information by ID
func (ps *PackageStore) GetPackageInfo(packageID PackageID) (*PackageInfo, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	pkg, exists := ps.packages[packageID]
	return pkg, exists
}

// GetFileByPath returns file ID by path
func (ps *PackageStore) GetFileByPath(filePath string) (FileID, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for id, file := range ps.files {
		if file.Path == filePath {
			return id, true
		}
	}
	return "", false
}

// GetAllFiles returns all registered files
func (ps *PackageStore) GetAllFiles() map[FileID]*FileInfo {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	result := make(map[FileID]*FileInfo)
	for id, file := range ps.files {
		result[id] = file
	}
	return result
}

// GetAllPackages returns all registered packages
func (ps *PackageStore) GetAllPackages() map[PackageID]*PackageInfo {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	result := make(map[PackageID]*PackageInfo)
	for id, pkg := range ps.packages {
		result[id] = pkg
	}
	return result
}

// GetFileMetadata returns file metadata for debug output
func (ps *PackageStore) GetFileMetadata(fileID FileID) *FileMetadata {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	file, exists := ps.files[fileID]
	if !exists {
		return nil
	}

	return &FileMetadata{
		URI:      fileURIForLocalPath(file.Path),
		Path:     file.Path,
		Filename: file.Filename,
	}
}

// extractFilenameFromPath extracts the filename from a file path
func extractFilenameFromPath(filePath string) string {
	// Simple implementation - can be enhanced for cross-platform support
	parts := strings.Split(filePath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return filePath
}

// extractPackageName extracts the package name from a package path
func extractPackageName(packagePath string) string {
	parts := strings.Split(packagePath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return packagePath
}
