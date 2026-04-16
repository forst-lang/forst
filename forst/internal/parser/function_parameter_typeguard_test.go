package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseFunction_destructuredParamWithShapeType(t *testing.T) {
	t.Parallel()
	src := `package main

func sum({a, b}: { a: Int, b: Int }): Int {
  return a
}
`
	log := ast.SetupTestLogger(nil)
	p := NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var fn ast.FunctionNode
	found := false
	for _, n := range nodes {
		if v, ok := n.(ast.FunctionNode); ok && string(v.Ident.ID) == "sum" {
			fn = v
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected function sum")
	}
	if len(fn.Params) != 1 {
		t.Fatalf("expected one param, got %d", len(fn.Params))
	}
	if _, ok := fn.Params[0].(ast.DestructuredParamNode); !ok {
		t.Fatalf("expected destructured param, got %T", fn.Params[0])
	}
}

func TestParseTypeGuard_inlineEnsureBody(t *testing.T) {
	t.Parallel()
	src := `package main

is (n Int) Positive ensure n is GreaterThan(0)

func main() {}
`
	log := ast.SetupTestLogger(nil)
	p := NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var guard *ast.TypeGuardNode
	for _, n := range nodes {
		switch v := n.(type) {
		case ast.TypeGuardNode:
			if string(v.Ident) == "Positive" {
				tmp := v
				guard = &tmp
			}
		case *ast.TypeGuardNode:
			if v != nil && string(v.Ident) == "Positive" {
				guard = v
			}
		}
	}
	if guard == nil {
		t.Fatal("expected Positive type guard")
	}
	if len(guard.Body) != 1 {
		t.Fatalf("expected inline guard body to lower to one ensure node, got %d", len(guard.Body))
	}
	if _, ok := guard.Body[0].(ast.EnsureNode); !ok {
		t.Fatalf("expected inline body ensure node, got %T", guard.Body[0])
	}
}

