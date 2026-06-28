package goload

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindModuleRoot_findsAncestor(t *testing.T) {
	root := t.TempDir()
	modDir := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testmod\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ft := filepath.Join(modDir, "x.ft")
	if err := os.WriteFile(ft, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := FindModuleRoot(ft); got != root {
		t.Fatalf("FindModuleRoot(file): got %q want %q", got, root)
	}
	if got := FindModuleRoot(modDir); got != root {
		t.Fatalf("FindModuleRoot(dir): got %q want %q", got, root)
	}
}

func TestModuleRootWithGoMod_findsAncestor(t *testing.T) {
	root := t.TempDir()
	modDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testmod\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ModuleRootWithGoMod(modDir)
	if err != nil {
		t.Fatalf("ModuleRootWithGoMod: %v", err)
	}
	if got != root {
		t.Fatalf("got %q want %q", got, root)
	}
}

func TestModuleRootWithGoMod_missingGoMod_errors(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ModuleRootWithGoMod(sub); err == nil {
		t.Fatal("expected error when no go.mod exists")
	}
}

func TestFindModuleRoot_noGoMod_fallsBackToStartDir(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := FindModuleRoot(sub); got != sub {
		t.Fatalf("got %q want %q", got, sub)
	}
}
