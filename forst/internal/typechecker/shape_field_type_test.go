package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestFieldTypeForNamedShape(t *testing.T) {
	t.Parallel()
	tc := New(setupTestLogger(nil), false)
	tc.Defs["Wrap"] = ast.TypeDefNode{
		Ident: "Wrap",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"r": {Type: &ast.TypeNode{Ident: ast.TypeResult, TypeParams: []ast.TypeNode{
						ast.NewBuiltinType(ast.TypeInt),
						ast.NewBuiltinType(ast.TypeError),
					}}},
				},
			},
		},
	}
	got, ok := tc.FieldTypeForNamedShape("Wrap", "r")
	if !ok || !got.IsResultType() {
		t.Fatalf("FieldTypeForNamedShape: ok=%v got=%v", ok, got)
	}
	if _, ok := tc.FieldTypeForNamedShape("Wrap", "missing"); ok {
		t.Fatal("expected false for missing field")
	}
}
