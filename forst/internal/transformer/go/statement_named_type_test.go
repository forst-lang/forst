package transformergo

import (
	"testing"

	"forst/internal/ast"
)

func TestFindBestNamedTypeForReturnType_nonHashReturnsName(t *testing.T) {
	t.Parallel()
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	got := tr.findBestNamedTypeForReturnType(ast.TypeNode{Ident: "User", TypeKind: ast.TypeKindBuiltin})
	if got != "User" {
		t.Fatalf("got %q", got)
	}
}

func TestFindBestNamedTypeForReturnType_hashNoCompatibleNamed_returnsEmpty(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	hash := ast.TypeNode{Ident: "T_isolatedHash", TypeKind: ast.TypeKindHashBased}
	got := tr.findBestNamedTypeForReturnType(hash)
	if got != "" {
		t.Fatalf("got %q want empty", got)
	}
}
