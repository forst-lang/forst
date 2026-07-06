package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"

	"github.com/sirupsen/logrus"
)

// NewTestLSPServer creates an LSP server backed by a temp module directory.
func NewTestLSPServer(tb testing.TB, module string) (*LSPServer, string) {
	tb.Helper()
	dir := tb.TempDir()
	if module == "" {
		module = "testmod"
	}
	testmod.WriteGoMod(tb, dir, module)
	log := logrus.New()
	log.SetOutput(os.Stderr)
	return NewLSPServer("0", log), dir
}

// OpenTestBuffer writes content to relPath under moduleRoot and registers it on the server.
func OpenTestBuffer(tb testing.TB, srv *LSPServer, moduleRoot, relPath, content string) string {
	tb.Helper()
	path := filepath.Join(moduleRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		tb.Fatal(err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		tb.Fatal(err)
	}
	uri := fileURIForLocalPath(abs)
	if srv.openDocuments == nil {
		srv.openDocuments = make(map[string]string)
	}
	srv.openDocuments[uri] = content
	return uri
}
