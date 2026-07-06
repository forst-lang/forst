package transformergo

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/typechecker"
	goast "go/ast"
)

func TestTransformFunction_destructuredParams(t *testing.T) {
	t.Parallel()
	src := `package main

func sum(a Int, {x, y}: { x: Int, y: Int }): Int {
  return x + y
}
`
	log := ast.SetupTestLogger(nil)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}

	tr := New(tc, log)
	var scopeNode ast.Node
	var fn ast.FunctionNode
	for _, n := range nodes {
		if f, ok := n.(ast.FunctionNode); ok && f.Ident.ID == "sum" {
			scopeNode = n
			fn = f
			break
		}
	}
	if scopeNode == nil {
		t.Fatal("sum not found")
	}

	decl, err := tr.transformFunction(scopeNode, fn)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}

	paramNames := []string{}
	for _, f := range decl.Type.Params.List {
		for _, n := range f.Names {
			paramNames = append(paramNames, n.Name)
		}
	}
	if len(paramNames) != 3 {
		t.Fatalf("expected 3 Go params, got %v", paramNames)
	}
	if paramNames[0] != "a" || paramNames[1] != "x" || paramNames[2] != "y" {
		t.Fatalf("unexpected param names: %v", paramNames)
	}
}

func TestTransformTypeGuard_typeLevelStub(t *testing.T) {
	t.Parallel()
	src := `package main

type MutationArg = Shape

is (m MutationArg) Input(input Shape) {
  ensure m is { input }
}
`
	log := ast.SetupTestLogger(nil)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}

	tr := New(tc, log)
	var guard ast.TypeGuardNode
	for _, n := range nodes {
		if g, ok := n.(ast.TypeGuardNode); ok && g.GetIdent() == "Input" {
			guard = g
			break
		}
		if gp, ok := n.(*ast.TypeGuardNode); ok && gp != nil && gp.GetIdent() == "Input" {
			guard = *gp
			break
		}
	}
	if guard.GetIdent() == "" {
		t.Fatal("expected Input type guard")
	}

	decl, err := tr.transformTypeGuard(guard)
	if err != nil {
		t.Fatalf("transformTypeGuard: %v", err)
	}
	if decl == nil {
		t.Fatal("expected type-level guard stub decl")
	}
	if !strings.HasPrefix(decl.Name.Name, "G_") {
		t.Fatalf("expected G_ prefix, got %s", decl.Name.Name)
	}
	if len(decl.Body.List) != 1 {
		t.Fatalf("expected stub body with single return, got %d stmts", len(decl.Body.List))
	}
	ret, ok := decl.Body.List[0].(*goast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		t.Fatalf("expected return stmt, got %#v", decl.Body.List[0])
	}
	if ident, ok := ret.Results[0].(*goast.Ident); !ok || ident.Name != "true" {
		t.Fatalf("expected return true, got %#v", ret.Results[0])
	}
}
