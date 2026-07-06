package compiler

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/internal/ast"
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

func TestCompileFile_shapeGuardExample(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	shapeGuardPath := filepath.Join(projectRoot, "examples", "in", "rfc", "guard", "shape_guard.ft")

	c := New(Args{
		Command:  "build",
		FilePath: shapeGuardPath,
		LogLevel: "error",
	}, nil)
	code, err := c.CompileFile()
	if err != nil {
		t.Fatalf("CompileFile(shape_guard.ft): %v", err)
	}
	if code == nil {
		t.Fatal("expected non-nil code")
	}
	out := *code
	if !strings.Contains(out, "func createTask(") {
		t.Fatal("expected createTask in shape_guard output")
	}
	if !strings.Contains(out, "func G_") {
		t.Fatal("expected G_ type guard function in shape_guard output")
	}
}

func TestTypecheckForCompileEntry_shapeGuardRebindsEntryScopes(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	shapeGuardPath := filepath.Join(projectRoot, "examples", "in", "rfc", "guard", "shape_guard.ft")

	c := New(Args{
		Command:  "run",
		FilePath: shapeGuardPath,
		LogLevel: "error",
	}, nil)

	entryNodes, err := c.loadInputNodesForCompile()
	if err != nil {
		t.Fatalf("loadInputNodesForCompile: %v", err)
	}
	tc, modResult, err := c.typecheckForCompile(entryNodes)
	if err != nil {
		t.Fatalf("typecheckForCompile: %v", err)
	}
	if modResult == nil {
		t.Fatal("expected module result")
	}

	var loggedInNode ast.Node
	for _, n := range entryNodes {
		switch g := n.(type) {
		case ast.TypeGuardNode:
			if g.Ident == "LoggedIn" {
				loggedInNode = n
			}
		case *ast.TypeGuardNode:
			if g != nil && g.Ident == "LoggedIn" {
				loggedInNode = n
			}
		}
	}
	if loggedInNode == nil {
		t.Fatal("LoggedIn not found in entry nodes")
	}
	if !tc.HasScopeForNode(loggedInNode) {
		t.Fatal("expected scope on entry LoggedIn node after rebind")
	}
	if err := tc.RestoreScope(loggedInNode); err != nil {
		t.Fatalf("RestoreScope entry LoggedIn: %v", err)
	}
}
