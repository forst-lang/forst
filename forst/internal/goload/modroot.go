package goload

import (
	"os"
	"path/filepath"
)

// FindModuleRoot walks upward from start (file or directory) until a directory
// containing go.mod is found. If none is found, it returns the directory that
// contained start (the starting folder for a directory, or the file's parent).
//
// This is used as go/packages Config.Dir so Forst imports resolve against the
// same module as the surrounding Go project (e.g. sidecar / monorepo roots).
func FindModuleRoot(start string) string {
	startDir := start
	if info, err := os.Stat(start); err == nil && !info.IsDir() {
		startDir = filepath.Dir(start)
	}
	startDir = filepath.Clean(startDir)
	dir := startDir
	for {
		if st, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !st.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return startDir
		}
		dir = parent
	}
}
