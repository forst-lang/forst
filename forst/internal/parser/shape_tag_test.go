package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseShapeType_fieldWithBacktickStructTag(t *testing.T) {
	t.Parallel()
	src := `package main

type Config = {
  host: String ` + "`json:\"host\"`" + `
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	def := nodes[1].(ast.TypeDefNode)
	shapeExpr := def.Expr.(ast.TypeDefShapeExpr)
	field := shapeExpr.Shape.Fields["host"]
	if field.Tag != `json:"host"` {
		t.Fatalf("host tag = %q, want json:\"host\"", field.Tag)
	}
}

func TestParseShapeType_fieldWithQuotedStructTag(t *testing.T) {
	t.Parallel()
	src := `package main

type Config = {
  port: Int "json:\"port,omitempty\""
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	def := nodes[1].(ast.TypeDefNode)
	shapeExpr := def.Expr.(ast.TypeDefShapeExpr)
	field := shapeExpr.Shape.Fields["port"]
	if field.Tag != `json:"port,omitempty"` {
		t.Fatalf("port tag = %q, want json:\"port,omitempty\"", field.Tag)
	}
}

func TestParseShapeType_fieldWithoutStructTag(t *testing.T) {
	t.Parallel()
	src := `package main

type Config = {
  name: String
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	def := nodes[1].(ast.TypeDefNode)
	shapeExpr := def.Expr.(ast.TypeDefShapeExpr)
	if shapeExpr.Shape.Fields["name"].Tag != "" {
		t.Fatalf("expected no tag, got %q", shapeExpr.Shape.Fields["name"].Tag)
	}
}
