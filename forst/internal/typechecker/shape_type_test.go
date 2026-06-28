package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestIsShapeType(t *testing.T) {
	tc := New(nil, false)

	if !tc.IsShapeType(ast.TypeNode{Ident: ast.TypeShape}) {
		t.Fatal("Shape builtin should be a shape type")
	}
	if tc.IsShapeType(ast.TypeNode{Ident: ast.TypeString}) {
		t.Fatal("String should not be a shape type")
	}

	tc.Defs["User"] = ast.TypeDefNode{
		Ident: "User",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"id": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}
	if !tc.IsShapeType(ast.TypeNode{Ident: "User"}) {
		t.Fatal("typedef to shape literal should be a shape type")
	}

	tc.Defs["AppContext"] = ast.TypeDefNode{
		Ident: "AppContext",
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: typeIdentPtr(string(ast.TypeShape)),
			},
		},
	}
	if !tc.IsShapeType(ast.TypeNode{Ident: "AppContext"}) {
		t.Fatal("typedef alias to Shape should be a shape type")
	}
}
