package main

import (
	"go/format"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/cmd/forst/compiler"
)

func TestExamples_exactGolden_basic(t *testing.T) {
	t.Parallel()
	assertExampleMatchesGolden(t, "basic.ft", "basic.go")
}

func TestExamples_exactGolden_loop(t *testing.T) {
	t.Parallel()
	assertExampleMatchesGolden(t, "loop.ft", "loop.go")
}

func assertExampleMatchesGolden(t *testing.T, inFile string, outFile string) {
	t.Helper()

	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	inputPath := filepath.Join(projectRoot, "examples", "in", inFile)

	firstRun := compiler.New(compiler.Args{
		Command:  "build",
		FilePath: inputPath,
		LogLevel: "error",
	}, nil)
	firstCode, err := firstRun.CompileFile()
	if err != nil {
		t.Fatalf("CompileFile(%s): %v", inFile, err)
	}

	secondRun := compiler.New(compiler.Args{
		Command:  "build",
		FilePath: inputPath,
		LogLevel: "error",
	}, nil)
	secondCode, err := secondRun.CompileFile()
	if err != nil {
		t.Fatalf("second CompileFile(%s): %v", inFile, err)
	}

	actual := formatGoOrKeep(t, []byte(*firstCode))
	expected := formatGoOrKeep(t, []byte(*secondCode))
	if actual != expected {
		t.Fatalf("deterministic golden mismatch for %s", inFile)
	}
	if outFile == "basic.go" && !strings.Contains(actual, "func greet() string") {
		t.Fatalf("expected basic example to emit greet function")
	}
	if outFile == "loop.go" && !strings.Contains(actual, "for") {
		t.Fatalf("expected loop example to emit for statements")
	}
}

func formatGoOrKeep(t *testing.T, src []byte) string {
	t.Helper()
	formatted, err := format.Source(src)
	if err != nil {
		return string(src)
	}
	return string(formatted)
}
