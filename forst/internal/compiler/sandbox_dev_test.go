package compiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSandboxModCache_skipsTidyWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	goMod := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cache := &SandboxModCache{}
	needs, err := cache.NeedsTidy(goMod)
	if err != nil || !needs {
		t.Fatalf("first tidy needed=%v err=%v", needs, err)
	}
	if err := cache.Record(goMod); err != nil {
		t.Fatal(err)
	}
	needs, err = cache.NeedsTidy(goMod)
	if err != nil || needs {
		t.Fatalf("second tidy should be skipped, needed=%v err=%v", needs, err)
	}
}

func TestDevSandboxDir_stablePath(t *testing.T) {
	root := t.TempDir()
	got := DevSandboxDir(root)
	want := filepath.Join(root, ".forst", "run", "dev")
	if got != want {
		t.Fatalf("DevSandboxDir=%q want %q", got, want)
	}
}

func TestIsDevBinPath(t *testing.T) {
	root := t.TempDir()
	bin := DevBinPath(root)
	if !IsDevBinPath(bin, root) {
		t.Fatal("expected dev bin path match")
	}
	if IsDevBinPath(filepath.Join(root, "main.go"), root) {
		t.Fatal("main.go should not match dev bin path")
	}
}
