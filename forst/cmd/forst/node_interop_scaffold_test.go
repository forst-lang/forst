package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/internal/compiler"
)

func TestNodeInterop_fixtureDirectoryExists(t *testing.T) {
	t.Parallel()
	dir := nodeInteropFixtureDir(t)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("bridge-interop fixture not yet created: %v", err)
	}
}

func TestNodeInterop_nodeInteropExampleRejectsRequireNoBridge(t *testing.T) {
	t.Parallel()
	dir := nodeInteropFixtureDir(t)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("bridge-interop fixture not yet created: %v", err)
	}
	entry := filepath.Join(dir, "main.ft")

	c := compiler.New(compiler.Args{
		Command:       "build",
		FilePath:      entry,
		PackageRoot:   dir,
		LogLevel:      "error",
		RequireNoBridge: true,
	}, nil)
	_, err := c.CompileFile()
	if err == nil {
		t.Fatal("expected bridge-interop compile to fail with -require-no-bridge")
	}
	if !strings.Contains(err.Error(), "require-no-bridge") {
		t.Fatalf("error = %v", err)
	}
}

func TestNodeInterop_basicExampleCompilesWithRequireNoBridge(t *testing.T) {
	t.Parallel()
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	inputPath := filepath.Join(projectRoot, "examples", "in", "basic.ft")
	src, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read basic.ft: %v", err)
	}
	// Compile from an isolated temp copy so examples/in module merge (e.g. bridge-interop) does not affect basic.ft.
	tmp := filepath.Join(t.TempDir(), "basic.ft")
	if err := os.WriteFile(tmp, src, 0o644); err != nil {
		t.Fatal(err)
	}

	c := compiler.New(compiler.Args{
		Command:       "build",
		FilePath:      tmp,
		LogLevel:      "error",
		RequireNoBridge: true,
	}, nil)
	code, err := c.CompileFile()
	if err != nil {
		t.Fatalf("CompileFile(basic.ft) with -require-no-bridge: %v", err)
	}
	if code == nil || !strings.Contains(*code, "func greet() string") {
		t.Fatal("expected basic example to compile unchanged")
	}
}

func nodeInteropFixtureDir(t *testing.T) string {
	t.Helper()
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	return filepath.Join(projectRoot, "examples", "in", "rfc", "bridge-interop")
}
