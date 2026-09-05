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
	exampleRoot := filepath.Join(projectRoot, "examples", "in", "rfc", "bridge-interop")
	entry := filepath.Join(exampleRoot, "main.ft")

	c := New(Args{
		Command:     "build",
		FilePath:    entry,
		PackageRoot: exampleRoot,
		LogLevel:    "error",
	}, nil)
	main, runtime, _, _, _, err := c.CompileWithBridgeRuntime()
	if err != nil {
		t.Fatalf("CompileWithBridgeRuntime: %v", err)
	}
	if !strings.Contains(main, "forst_bridge_callsync_") {
		t.Fatalf("missing bridge wrapper in main:\n%s", main)
	}
	if runtime == "" || !strings.Contains(runtime, "bridgert.CallSync") {
		t.Fatalf("missing bridgert.CallSync in runtime:\n%s", runtime)
	}
	if !strings.Contains(runtime, "forstBridgeManifestJSON") {
		t.Fatalf("missing manifest in runtime:\n%s", runtime)
	}
}
