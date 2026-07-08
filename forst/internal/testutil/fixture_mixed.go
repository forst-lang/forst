package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"
)

const mixedGoHelpersSource = `package memos

func Add(a, b int) int {
	return a + b
}

func unexported() int {
	return 0
}

func OpenValue() (int, error) {
	return 42, nil
}

// StringFromBytes converts memo file bytes to a string for mixed-package tests.
func StringFromBytes(b []byte) string {
	return string(b)
}
`

var (
	mixedGoMkdirAll  = os.MkdirAll
	mixedGoWriteFile = os.WriteFile
)

// WriteMixedGoForstModule creates a temp module with mixedtest/<module>/helpers.go.
func WriteMixedGoForstModule(tb testing.TB, module string) (root, importPath string) {
	tb.Helper()
	if module == "" {
		module = "memos"
	}
	root = tb.TempDir()
	modName := "mixedtest"
	testmod.WriteGoMod(tb, root, modName)
	pkgDir := filepath.Join(root, module)
	if err := mixedGoMkdirAll(pkgDir, 0o755); err != nil {
		tbFail(tb, err)
		return "", ""
	}
	if err := mixedGoWriteFile(filepath.Join(pkgDir, "helpers.go"), []byte(mixedGoHelpersSource), 0o644); err != nil {
		tbFail(tb, err)
		return "", ""
	}
	return root, modName + "/" + module
}
