// Package safefs provides symlink-safe filesystem access confined to a directory root.
package safefs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	safefsPathAbs = filepath.Abs
	safefsPathRel = filepath.Rel
	safefsOpenRoot = os.OpenRoot
)

// RootedFS confines file operations to one directory tree using os.Root.
type RootedFS struct {
	root    *os.Root
	absRoot string
}

// OpenRoot opens absRoot as the filesystem root. absRoot must be an existing directory.
func OpenRoot(absRoot string) (*RootedFS, error) {
	abs, err := safefsPathAbs(absRoot)
	if err != nil {
		return nil, err
	}
	abs = filepath.Clean(abs)
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("safefs: not a directory: %s", abs)
	}
	root, err := safefsOpenRoot(abs)
	if err != nil {
		return nil, err
	}
	return &RootedFS{root: root, absRoot: abs}, nil
}

// AbsRoot returns the absolute root directory path.
func (r *RootedFS) AbsRoot() string {
	return r.absRoot
}

// Close releases the root.
func (r *RootedFS) Close() error {
	return r.root.Close()
}

// FS returns an fs.FS for the tree under the root (symlink-safe walks).
func (r *RootedFS) FS() fs.FS {
	return r.root.FS()
}

// ReadFile reads a file relative to the root.
func (r *RootedFS) ReadFile(rel string) ([]byte, error) {
	return fs.ReadFile(r.root.FS(), filepath.ToSlash(rel))
}

// Stat returns file info for a path relative to the root.
func (r *RootedFS) Stat(rel string) (fs.FileInfo, error) {
	return fs.Stat(r.root.FS(), filepath.ToSlash(rel))
}

// RelPath returns a slash-separated relative path when absPath is under the root.
func (r *RootedFS) RelPath(absPath string) (string, error) {
	abs, err := safefsPathAbs(absPath)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	rel, err := safefsPathRel(r.absRoot, abs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside root %q", abs, r.absRoot)
	}
	return filepath.ToSlash(rel), nil
}

// AbsPath joins absRoot with a slash-separated relative path from FS walks.
func (r *RootedFS) AbsPath(rel string) string {
	return filepath.Join(r.absRoot, filepath.FromSlash(rel))
}
