package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"
)

const mixedGoHelpersSource = `package mixed

func Add(a, b int) int {
	return a + b
}

func unexported() int {
	return 0
}

func OpenValue() (int, error) {
	return 42, nil
}
`

// WriteMixedGoForstModule creates a temp module with mixedtest/<module>/helpers.go.
func WriteMixedGoForstModule(tb testing.TB, module string) (root, importPath string) {
	tb.Helper()
	if module == "" {
		module = "mixed"
	}
	root = tb.TempDir()
	modName := "mixedtest"
	testmod.WriteGoMod(tb, root, modName)
	mixedDir := filepath.Join(root, module)
	if err := os.MkdirAll(mixedDir, 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mixedDir, "helpers.go"), []byte(mixedGoHelpersSource), 0o644); err != nil {
		tb.Fatal(err)
	}
	return root, modName + "/" + module
}
