package goload

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModulePath_readsGoMod(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ModulePath(dir); got != "example.com/test" {
		t.Fatalf("ModulePath = %q", got)
	}
}
