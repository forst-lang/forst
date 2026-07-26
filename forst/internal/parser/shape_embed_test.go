package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseShapeType_embeddedField(t *testing.T) {
	t.Parallel()
	src := `package main

type Inner = {
  Value: Int
}

type Outer = {
  Inner
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	outerDef := nodes[2].(ast.TypeDefNode)
	shapeExpr, ok := outerDef.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("want shape typedef, got %T", outerDef.Expr)
	}
	field, ok := shapeExpr.Shape.Fields["Inner"]
	if !ok || !field.Embedded {
		t.Fatalf("Inner field = %#v ok=%v", field, ok)
	}
}

func TestParseShapeType_namedFieldNotEmbedded(t *testing.T) {
	t.Parallel()
	src := `package main

type Outer = {
  inner: Inner
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	outerDef := nodes[1].(ast.TypeDefNode)
	shapeExpr := outerDef.Expr.(ast.TypeDefShapeExpr)
	field := shapeExpr.Shape.Fields["inner"]
	if field.Embedded {
		t.Fatal("explicit inner: Inner should not be embedded")
	}
}
