package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunGoSourceFiles_missingGoFiles(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(outPath, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	emptyDir := filepath.Join(dir, "empty")
	if err := os.Mkdir(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runGoSourceFiles(filepath.Join(emptyDir, "main.go")); err == nil {
		t.Fatal("expected error when sandbox has no go files")
	}
}

func TestSetRunEnvBoundaryRoot_setsForstRoot(t *testing.T) {
	root := t.TempDir()
	env := setRunEnvBoundaryRoot([]string{"FORST_ROOT=/old", "PATH=/bin"}, root)
	found := false
	for _, entry := range env {
		if entry == "FORST_ROOT="+root {
			found = true
		}
		if entry == "FORST_ROOT=/old" {
			t.Fatal("expected old FORST_ROOT to be replaced")
		}
	}
	if !found {
		t.Fatalf("FORST_ROOT not set in env: %#v", env)
	}
}

func TestAppendRunEnvVar_replacesExistingKey(t *testing.T) {
	env := appendRunEnvVar([]string{"CGO_ENABLED=1", "PATH=/bin"}, "CGO_ENABLED", "0")
	count := 0
	for _, entry := range env {
		if strings.HasPrefix(entry, "CGO_ENABLED=") {
			count++
			if entry != "CGO_ENABLED=0" {
				t.Fatalf("CGO_ENABLED = %q want 0", entry)
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected one CGO_ENABLED entry, got %d in %#v", count, env)
	}
}

func TestBuildGoProgramInSandboxWithTarget_crossCompileSetsCGOEnabledZero(t *testing.T) {
	goos := "linux"
	if runtime.GOOS == "linux" {
		goos = "windows"
	}
	env := os.Environ()
	env = appendRunEnvVar(env, "GOOS", goos)
	if (goos != runtime.GOOS) {
		env = appendRunEnvVar(env, "CGO_ENABLED", "0")
	}
	found := false
	for _, entry := range env {
		if entry == "CGO_ENABLED=0" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected CGO_ENABLED=0 for cross-compile env, got %#v", env)
	}
}
