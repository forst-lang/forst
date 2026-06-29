package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func TestDestructuredParam_endToEnd(t *testing.T) {
	t.Parallel()
	src := `package main

func sum({a, b}: { a: Int, b: Int }): Int {
  return a + b
}
`
	log := ast.SetupTestLogger(nil)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}

	sig, ok := tc.Functions[ast.Identifier("sum")]
	if !ok || len(sig.Parameters) != 1 {
		t.Fatalf("expected one parameter signature for sum, got %v", sig.Parameters)
	}
}

func TestShapeFieldsFromParamType_inlineShape(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	baseType := ast.TypeIdent(ast.TypeShape)
	typeNode := ast.TypeNode{
		Ident: ast.TypeShape,
		Assertion: &ast.AssertionNode{
			BaseType: &baseType,
			Constraints: []ast.ConstraintNode{{
				Name: ConstraintMatch,
				Args: []ast.ConstraintArgumentNode{{
					Shape: &ast.ShapeNode{
						Fields: map[string]ast.ShapeFieldNode{
							"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
						},
					},
				}},
			}},
		},
	}
	fields, ok := tc.ShapeFieldsFromParamType(typeNode)
	if !ok || len(fields) != 1 {
		t.Fatalf("ShapeFieldsFromParamType: ok=%v fields=%d", ok, len(fields))
	}
}

func TestShapeFieldsFromParamType_typedefShape(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	tc.registerType(ast.TypeDefNode{
		Ident: "Point",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
					"y": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	})
	fields, ok := tc.ShapeFieldsFromParamType(ast.TypeNode{Ident: "Point"})
	if !ok || len(fields) != 2 {
		t.Fatalf("typedef shape fields: ok=%v len=%d", ok, len(fields))
	}
}

func TestShapeFieldTypeNode_nestedShape(t *testing.T) {
	t.Parallel()
	field := ast.ShapeFieldNode{
		Shape: &ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"z": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			},
		},
	}
	ty, ok := ShapeFieldTypeNode(field)
	if !ok || ty.Ident != ast.TypeShape || ty.Assertion == nil {
		t.Fatalf("nested shape field type: %#v ok=%v", ty, ok)
	}
}
