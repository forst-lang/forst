package transformergo

import (
	"testing"

	"forst/internal/ast"
)

func TestFindBestNamedTypeForReturnStructLiteral_fallbackToExpectedWhenNoNamedStructuralMatch(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	shape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"z": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
		},
	}
	exp := &ast.TypeNode{Ident: "T_only_fallback", TypeKind: ast.TypeKindHashBased}
	got := tr.findBestNamedTypeForReturnStructLiteral(shape, exp)
	if got == nil || got.Ident != exp.Ident {
		t.Fatalf("want %+v, got %+v", exp, got)
	}
}
