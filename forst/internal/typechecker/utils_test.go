package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestIsPlainFailureCompatibleWithDeclaredResult(t *testing.T) {
	t.Parallel()
	tc := New(setupTestLogger(nil), false)
	result := ast.TypeNode{
		Ident:      ast.TypeResult,
		TypeParams: []ast.TypeNode{{Ident: ast.TypeInt}, {Ident: ast.TypeString}},
	}
	if !tc.isPlainFailureCompatibleWithDeclaredResult(
		ast.TypeNode{Ident: ast.TypeString},
		result,
	) {
		t.Fatal("plain string failure should match Result(Int, String)")
	}
	if tc.isPlainFailureCompatibleWithDeclaredResult(
		ast.TypeNode{Ident: ast.TypeInt},
		result,
	) {
		t.Fatal("int should not match failure slot String")
	}
}

func TestReturnTypeMatchesInferredShape(t *testing.T) {
	t.Parallel()
	tc := New(setupTestLogger(nil), false)
	hashName := ast.TypeIdent("T_abc")
	tc.Defs[hashName] = ast.TypeDefNode{
		Ident: hashName,
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	}
	base := ast.TypeShape
	expected := ast.TypeNode{
		Ident: ast.TypeShape,
		Assertion: &ast.AssertionNode{
			BaseType: &base,
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
	if !tc.returnTypeMatchesInferredShape(ast.TypeNode{Ident: hashName}, expected) {
		t.Fatal("hash shape should match parsed Match return type")
	}
}

func TestRegisterHashBasedType_idempotent(t *testing.T) {
	t.Parallel()
	tc := New(setupTestLogger(nil), false)
	fields := map[string]ast.ShapeFieldNode{
		"a": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
	}
	RegisterHashBasedType(tc, "H1", fields)
	RegisterHashBasedType(tc, "H1", map[string]ast.ShapeFieldNode{
		"b": {Type: &ast.TypeNode{Ident: ast.TypeString}},
	})
	if def, ok := tc.Defs["H1"].(ast.TypeDefNode); !ok || len(def.Expr.(ast.TypeDefShapeExpr).Shape.Fields) != 1 {
		t.Fatal("second register should not overwrite")
	}
}
