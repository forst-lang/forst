package compiler

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompileNodeInteropExample(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	exampleRoot := filepath.Join(projectRoot, "examples", "in", "rfc", "node-interop")
	entry := filepath.Join(exampleRoot, "main.ft")

	c := New(Args{
		Command:     "build",
		FilePath:    entry,
		PackageRoot: exampleRoot,
		LogLevel:    "error",
	}, nil)
	main, runtime, _, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("CompileWithNodeRuntime: %v", err)
	}
	if !strings.Contains(main, "forst_node_callsync_") {
		t.Fatalf("missing bridge wrapper in main:\n%s", main)
	}
	if runtime == "" || !strings.Contains(runtime, "nodert.CallSync") {
		t.Fatalf("missing nodert.CallSync in runtime:\n%s", runtime)
	}
	if !strings.Contains(runtime, "forstNodeManifestJSON") {
		t.Fatalf("missing manifest in runtime:\n%s", runtime)
	}
}
