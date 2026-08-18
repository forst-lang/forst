package parser

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
)

func TestParseTypeDef_shapeWithTrailingRouter(t *testing.T) {
	src := `package catalog

type Catalog = {
  PlaceOrder(sku String): Result(String, Error)
}.Router()
`
	l := lexer.New([]byte(src), "test.ft", nil)
	tokens := l.Lex()
	p := New(tokens, "test.ft", nil)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var td *ast.TypeDefNode
	for _, n := range nodes {
		if d, ok := n.(ast.TypeDefNode); ok && d.Ident == "Catalog" {
			td = &d
			break
		}
	}
	if td == nil {
		t.Fatal("Catalog typedef not found")
	}
	ae, ok := td.Expr.(ast.TypeDefAssertionExpr)
	if !ok || ae.Assertion == nil {
		t.Fatalf("expected TypeDefAssertionExpr, got %T", td.Expr)
	}
	foundRouter := false
	for _, c := range ae.Assertion.Constraints {
		if c.Name == "Router" {
			foundRouter = true
		}
	}
	if !foundRouter {
		t.Fatalf("Router constraint not found: %#v", ae.Assertion.Constraints)
	}
}
