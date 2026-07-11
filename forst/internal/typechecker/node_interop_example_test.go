package typechecker

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/parser"
	"forst/internal/testutil"
)

func TestNodeInteropExampleFile_typechecks(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	entry := filepath.Join(projectRoot, "examples", "in", "rfc", "node-interop", "main.ft")
	src, err := os.ReadFile(entry)
	if err != nil {
		t.Fatal(err)
	}
	log := testutil.TestLogger(t, nil)
	p := parser.NewTestParser(string(src), log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, n := range nodes {
		if imp, ok := n.(ast.ImportNode); ok {
			t.Logf("import NodeOptIn=%v path=%q", imp.NodeOptIn, imp.Path)
		}
	}
	tc := New(log, false)
	exampleRoot := filepath.Dir(entry)
	absRoot, err := filepath.Abs(exampleRoot)
	if err != nil {
		t.Fatal(err)
	}
	tc.NodeBoundaryRoot = absRoot
	goModRoot := goload.FindModuleRoot(exampleRoot)
	tc.ConfigureForForstFile(goModRoot, exampleRoot, nodes)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	if !tc.NeedsNodeRuntime() {
		t.Fatal("expected NeedsNodeRuntime")
	}
	_ = forstpkg.PackageNameFromNodes(nodes)
}
