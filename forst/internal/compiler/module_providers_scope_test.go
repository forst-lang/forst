package compiler

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestModuleRootForProvidersPass_singleExampleFile(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	basicPath := filepath.Join(projectRoot, "examples", "in", "basic.ft")

	c := New(Args{
		Command:  "build",
		FilePath: basicPath,
		LogLevel: "error",
	}, nil)

	root := c.moduleRootForProvidersPass()
	want := filepath.Join(projectRoot, "examples", "in")
	if root != want {
		t.Fatalf("moduleRootForProvidersPass() = %q, want %q", root, want)
	}
}

func TestCompileFile_basicExampleNoModuleWalkPanic(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	basicPath := filepath.Join(projectRoot, "examples", "in", "basic.ft")

	c := New(Args{
		Command:  "build",
		FilePath: basicPath,
		LogLevel: "error",
	}, nil)
	code, err := c.CompileFile()
	if err != nil {
		t.Fatalf("CompileFile(basic.ft): %v", err)
	}
	if code == nil || !strings.Contains(*code, "func greet()") {
		t.Fatalf("expected greet in output")
	}
}
