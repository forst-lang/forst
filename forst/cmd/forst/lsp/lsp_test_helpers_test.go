package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
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

var sharedImportTestModule struct {
	once sync.Once
	dir  string
	err  error
}

var sharedImportTestMu sync.Mutex

// sharedImportTestDir returns a module root with go.mod, reused across import/hover tests
// so go/packages loads are cached via goload.LoadByPkgPath.
func sharedImportTestDir(t *testing.T) string {
	t.Helper()
	sharedImportTestModule.once.Do(func() {
		sharedImportTestModule.dir, sharedImportTestModule.err = os.MkdirTemp("", "lsp-shared-import-*")
		if sharedImportTestModule.err != nil {
			return
		}
		sharedImportTestModule.err = os.WriteFile(
			filepath.Join(sharedImportTestModule.dir, "go.mod"),
			[]byte("module lsp_import_test\n\ngo 1.23\n"),
			0o644,
		)
	})
	if sharedImportTestModule.err != nil {
		t.Fatal(sharedImportTestModule.err)
	}
	return sharedImportTestModule.dir
}

func cleanSharedImportFtFiles(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ft") {
			continue
		}
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
}

// sharedImportTestFile writes content under the shared import test module and returns path + file URI.
// Access is serialized so parallel tests do not merge each other's on-disk .ft peers.
func sharedImportTestFile(t *testing.T, baseName, content string) (path, uri string) {
	t.Helper()
	sharedImportTestMu.Lock()
	t.Cleanup(func() {
		if path != "" {
			_ = os.Remove(path)
		}
		sharedImportTestMu.Unlock()
	})
	dir := sharedImportTestDir(t)
	cleanSharedImportFtFiles(dir)
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
