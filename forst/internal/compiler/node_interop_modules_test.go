package compiler_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"forst/internal/ast"
	"forst/internal/compiler"
	"forst/internal/forstpkg"
)

func TestNodeInteropModules_typechecksNestedCheckoutEntry(t *testing.T) {
	root := nodeInteropModulesDir(t)
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
		FilePath:           mainPath,
		PackageRoot:        root,
		ExportStructFields: true,
		LogLevel:           "error",
	}, nil)
	main, runtime, _, _, _, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("CompileWithNodeRuntime: %v", err)
	}
	if main == "" || !containsAll(main, "forst_node_callsync_") {
		t.Fatalf("generated main missing bridge wrapper:\n%s", main)
	}
	if runtime == "" || !containsAll(runtime,
		"nodert.CallSync",
		"forstNodeManifestJSON",
		"legacy/api/checkout.ts",
		"createOrder",
	) {
		t.Fatalf("generated runtime missing nested module wiring:\n%s", runtime)
	}
}

func nodeInteropModulesDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "examples", "in", "rfc", "node-interop", "modules"))
}
