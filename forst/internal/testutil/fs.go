package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"
)

// WriteFile writes content to path, creating parent directories as needed.
func WriteFile(tb testing.TB, path, content string) {
	tb.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		tb.Fatal(err)
	}
}

// WriteModule creates a temp module with go.mod and the given relative file paths.
func WriteModule(tb testing.TB, module string, files map[string]string) string {
	tb.Helper()
	dir := tb.TempDir()
	if module == "" {
		module = "testmod"
	}
	WriteFile(tb, filepath.Join(dir, "go.mod"), testmod.GoModContent(module))
	for rel, content := range files {
		WriteFile(tb, filepath.Join(dir, rel), content)
	}
	return dir
}

// WriteForst writes a .ft file under moduleRoot and returns its absolute path.
func WriteForst(tb testing.TB, moduleRoot, relPath, src string) string {
	tb.Helper()
	path := filepath.Join(moduleRoot, relPath)
	WriteFile(tb, path, src)
	abs, err := filepath.Abs(path)
	if err != nil {
		tb.Fatal(err)
	}
	return abs
}

// FileURI returns a stable file:// URI for an absolute path.
func FileURI(tb testing.TB, absPath string) string {
	tb.Helper()
	abs, err := filepath.Abs(absPath)
	if err != nil {
		tb.Fatal(err)
	}
	if abs == "" {
		tb.Fatal("empty path")
	}
	// Match lsp.fileURIForLocalPath behavior without importing cmd/forst/lsp.
	return "file://" + abs
}
