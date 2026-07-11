package compiler_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"forst/internal/ast"
	"forst/internal/compiler"
	"forst/internal/forstpkg"
)

func TestNodeInteropSync_typechecksWithPackageRoot(t *testing.T) {
	root := nodeInteropSyncDir(t)
	mainPath := filepath.Join(root, "main.ft")
	paths, err := collectSamePackagePathsForTest(root, mainPath)
	if err != nil {
		t.Fatalf("collect paths: %v", err)
	}
	merged, _, err := forstpkg.ParseAndMergePackage(nil, paths)
	if err != nil {
		t.Fatalf("parse merge: %v", err)
	}
	for _, n := range merged {
		if imp, ok := n.(ast.ImportNode); ok && imp.NodeOptInSource != "" && !imp.NodeOptIn {
			t.Fatalf("node import missing NodeOptIn: %+v", imp)
		}
	}

	c := compiler.New(compiler.Args{
		Command:            "build",
		FilePath:           filepath.Join(root, "main.ft"),
		PackageRoot:        root,
		ExportStructFields: true,
		LogLevel:           "error",
	}, nil)
	main, runtime, _, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("CompileWithNodeRuntime: %v", err)
	}
	if main == "" || !containsAll(main, "forst_node_callsync_", "resultErr") {
		t.Fatalf("generated main missing bridge wrapper:\n%s", main)
	}
	if runtime == "" || !containsAll(runtime, "nodert.CallSync", "forstNodeManifestJSON") {
		t.Fatalf("generated runtime missing nodert wiring:\n%s", runtime)
	}
}

func nodeInteropSyncDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "examples", "in", "rfc", "node-interop", "sync"))
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
func collectSamePackagePathsForTest(_ string, entryPath string) ([]string, error) {
	return []string{entryPath}, nil
}
