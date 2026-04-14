package compiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWatchFile_rejectsPackageRoot(t *testing.T) {
	c := New(Args{
		FilePath:    filepath.Join(t.TempDir(), "entry.ft"),
		PackageRoot: t.TempDir(),
	}, nil)
	err := c.WatchFile()
	if err == nil {
		t.Fatal("expected -watch with -root error")
	}
}

func TestCompileAndRunOnce_compileFailureDoesNotPanic(t *testing.T) {
	c := New(Args{
		Command:  "build",
		FilePath: filepath.Join(t.TempDir(), "missing.ft"),
		LogLevel: "error",
	}, nil)
	// should log and return without panic
	c.compileAndRunOnce()
}

func TestRunCompiledOutput_usesExplicitOutputPath(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "main.go")
	goCode := "package main\nfunc main() {}\n"
	if err := os.WriteFile(goFile, []byte(goCode), 0644); err != nil {
		t.Fatal(err)
	}

	c := New(Args{
		OutputPath: goFile,
	}, nil)
	if err := c.runCompiledOutput("ignored because output path is set"); err != nil {
		t.Fatalf("runCompiledOutput: %v", err)
	}
}
