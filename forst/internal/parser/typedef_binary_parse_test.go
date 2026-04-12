package parser

import (
	"testing"

	"forst/internal/ast"
)

// Regression: simple identifier on the left of | must still parse as TypeDefBinaryExpr (not return early).
func TestParseTypeDefExpr_simpleIdentUnion(t *testing.T) {
	t.Parallel()
	src := `package main

type A = Int

type U = A | String

func main() {}
`
	log := ast.SetupTestLogger(nil)
	p := NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	var td *ast.TypeDefNode
	for _, n := range nodes {
		if d, ok := n.(ast.TypeDefNode); ok && d.Ident == "U" {
			td = &d
			break
		}
	}
	if td == nil {
		t.Fatal("expected type U")
	}
	bin, ok := td.Expr.(ast.TypeDefBinaryExpr)
	if !ok {
		t.Fatalf("expected TypeDefBinaryExpr, got %T", td.Expr)
	}
	if !bin.IsDisjunction() {
		t.Fatal("expected disjunction")
	}
}

// TypeScript-style leading | after = (newlines are not tokens; lexer skips them).
func TestParseTypeDefExpr_leadingPipeUnion(t *testing.T) {
	t.Parallel()
	src := `package main

type A = Int

type U =
| A | String

func main() {}
`
	log := ast.SetupTestLogger(nil)
	p := NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	var td *ast.TypeDefNode
	for _, n := range nodes {
		if d, ok := n.(ast.TypeDefNode); ok && d.Ident == "U" {
			td = &d
			break
		}
	}
	if td == nil {
		t.Fatal("expected type U")
	}
	bin, ok := td.Expr.(ast.TypeDefBinaryExpr)
	if !ok {
		t.Fatalf("expected TypeDefBinaryExpr, got %T", td.Expr)
	}
	if !bin.IsDisjunction() {
		t.Fatal("expected disjunction")
	}
}
