package lsp

import (
	"path/filepath"
	"testing"
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
