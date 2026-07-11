package nodert

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveTsxLoaderPath_findsMonorepoRoot(t *testing.T) {
	root := repoRootForTsxTest(t)
	path, err := ResolveTsxLoaderPath(root)
	if err != nil {
		t.Fatalf("ResolveTsxLoaderPath: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("loader missing at %s: %v", path, err)
	}
	if filepath.Base(path) != "loader.mjs" {
		t.Fatalf("path = %q, want loader.mjs", path)
	}
}

func TestResolveTsxLoaderPath_findsFromBootstrapDir(t *testing.T) {
	root := repoRootForTsxTest(t)
	bootstrap := filepath.Join(root, "packages", "node-runtime", "dist", "bootstrap.js")
	path, err := ResolveTsxLoaderPath(bootstrap)
	if err != nil {
		t.Fatalf("ResolveTsxLoaderPath: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("loader missing: %v", err)
	}
}

func repoRootForTsxTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
