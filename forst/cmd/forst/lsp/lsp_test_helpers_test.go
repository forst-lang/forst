package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/testmod"
)

// mustFileURI returns a stable file:// URI for a path (typically under t.TempDir()).
func mustFileURI(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return fileURIForLocalPath(abs)
}

// importTestModuleFile writes content in an isolated temp module (go.mod + one .ft file).
// Safe for t.Parallel() unlike sharedImportTestFile which serializes a single shared directory.
func importTestModuleFile(t *testing.T, baseName, content string) (path, uri string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("lsp_import_test")), 0o644); err != nil {
		t.Fatal(err)
	}
	path = filepath.Join(dir, baseName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path, mustFileURI(t, path)
}

func sharedImportTestFileName(t *testing.T, suffix string) string {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	return name + suffix
}
