package typechecker_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/testutil"
	"forst/internal/typechecker"
)

func TestNodeInteropSyncExample_parsesNodeOptIn(t *testing.T) {
	root := nodeInteropSyncDir(t)
	nodes, err := forstpkg.ParseForstFile(testutil.TestLogger(t, nil), filepath.Join(root, "main.ft"))
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, n := range nodes {
		if imp, ok := n.(ast.ImportNode); ok {
			found = true
			if !imp.NodeOptIn {
				t.Fatalf("expected NodeOptIn on import, got %+v", imp)
			}
		}
	}
	if !found {
		t.Fatal("no ImportNode in parsed AST")
	}
}

func TestNodeInteropSyncExample_typechecks(t *testing.T) {
	root := nodeInteropSyncDir(t)
	src, err := os.ReadFile(filepath.Join(root, "main.ft"))
	if err != nil {
		t.Fatal(err)
	}
	tc, _ := typechecker.MustTypecheck(t, string(src), testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})
	if !tc.NeedsNodeRuntime() {
		t.Fatal("expected NeedsNodeRuntime")
	}
}

func nodeInteropSyncDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "examples", "in", "rfc", "node-interop", "sync"))
}
