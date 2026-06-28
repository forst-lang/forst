package goload

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"
)

func TestModulePath_readsGoMod(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("example.com/test")), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ModulePath(dir); got != "example.com/test" {
		t.Fatalf("ModulePath = %q", got)
	}
}
