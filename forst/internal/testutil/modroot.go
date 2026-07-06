package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/goload"
)

// ModuleRoot walks upward from the process cwd until go.mod is found.
func ModuleRoot(tb testing.TB) string {
	tb.Helper()
	dir, err := os.Getwd()
	if err != nil {
		tb.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			tb.Fatal("go.mod not found from cwd")
		}
		dir = parent
	}
}

// ModuleRootFrom returns the module root for start using goload.FindModuleRoot.
func ModuleRootFrom(tb testing.TB, start string) string {
	tb.Helper()
	root := goload.FindModuleRoot(start)
	if root == "" {
		tb.Fatalf("module root not found from %q", start)
	}
	return root
}

// ExamplePath resolves examples/in/<rel> from any package test working directory.
func ExamplePath(tb testing.TB, rel string) string {
	tb.Helper()
	candidates := []string{
		filepath.Join("..", "..", "..", "examples", "in", rel),
		filepath.Join("..", "..", "examples", "in", rel),
		filepath.Join("examples", "in", rel),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			abs, err := filepath.Abs(path)
			if err != nil {
				tb.Fatal(err)
			}
			return abs
		}
	}
	tb.Fatalf("example file not found: %s (tried %v)", rel, candidates)
	return ""
}
