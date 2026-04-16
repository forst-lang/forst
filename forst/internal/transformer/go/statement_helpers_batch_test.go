package transformergo

import (
	"testing"

	"forst/internal/ast"
)

func TestFindBestNamedTypeForReturnStructLiteral_prefersStructuralNamedType(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tc.Defs["User"] = ast.TypeDefNode{
		Ident: "User",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}
	tr := setupTransformer(tc, log)
	got := tr.findBestNamedTypeForReturnStructLiteral(
		ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			},
		},
		&ast.TypeNode{Ident: "T_hash", TypeKind: ast.TypeKindHashBased},
	)
	if got == nil || got.Ident != "User" {
		t.Fatalf("expected User named type, got %+v", got)
	}
}

func TestBuildCompositeLiteralForReturn_andWrapVariableInNamedStruct(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tc.Defs["Box"] = ast.TypeDefNode{
		Ident: "Box",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"value": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}
	tr := setupTransformer(tc, log)
	expr := tr.buildCompositeLiteralForReturn(&ast.TypeNode{Ident: "Box"}, "value")
	if expr == nil {
		t.Fatal("expected composite literal expr")
	}
	_, err := tr.wrapVariableInNamedStruct(&ast.TypeNode{Ident: "Box"}, ast.VariableNode{Ident: ast.Ident{ID: "value"}})
	if err != nil {
		t.Fatalf("wrapVariableInNamedStruct: %v", err)
	}
}

func TestGetAssertionStringForError_fallbackAndInferred(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	base := ast.TypeIdent("Int")
	a := &ast.AssertionNode{
		BaseType: &base,
		Constraints: []ast.ConstraintNode{{Name: "Required"}},
	}
	if got := tr.getAssertionStringForError(a); got == "" {
		t.Fatal("expected assertion string")
	}
	// nil base-type assertion exercises fallback to String()
	noBase := &ast.AssertionNode{Constraints: []ast.ConstraintNode{{Name: "Required"}}}
	_ = tr.getAssertionStringForError(noBase)
}

