package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/goload"
)

var (
	moduleRootGetwd   = os.Getwd
	examplePathAbs    = filepath.Abs
	moduleRootFromFind = goload.FindModuleRoot
)

// ModuleRoot walks upward from the process cwd until go.mod is found.
func ModuleRoot(tb testing.TB) string {
	tb.Helper()
	dir, err := moduleRootGetwd()
	if err != nil {
		tbFail(tb, err)
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			tbFail(tb, "go.mod not found from cwd")
			return ""
		}
		dir = parent
	}
}

// ModuleRootFrom returns the module root for start using goload.FindModuleRoot.
func ModuleRootFrom(tb testing.TB, start string) string {
	tb.Helper()
	root := moduleRootFromFind(start)
	if root == "" {
		tbFailf(tb, "module root not found from %q", start)
		return ""
	}
	return root
}

// ExamplePath resolves examples/in/<rel> from any package test working directory.
func ExamplePath(tb testing.TB, rel string) string {
	tb.Helper()
	candidates := []string{
		filepath.Join("..", "..", "..", "..", "examples", "in", rel),
		filepath.Join("..", "..", "..", "examples", "in", rel),
		filepath.Join("..", "..", "examples", "in", rel),
		filepath.Join("examples", "in", rel),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			abs, err := examplePathAbs(path)
			if err != nil {
				tbFail(tb, err)
				return ""
			}
			return abs
		}
	}
	tbFailf(tb, "example file not found: %s (tried %v)", rel, candidates)
	return ""
}
