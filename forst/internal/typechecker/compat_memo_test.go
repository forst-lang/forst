package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestIsTypeCompatible_memoizesRepeatedPairs(t *testing.T) {
	t.Parallel()
	tc := testTypeChecker(t)
	tc.compatMemo = make(map[compatKey]bool)
	actual := ast.TypeNode{Ident: ast.TypeInt}
	expected := ast.TypeNode{Ident: ast.TypeInt}
	if !tc.IsTypeCompatible(actual, expected) {
		t.Fatal("Int should be compatible with Int")
	}
	key := compatKeyFor(actual, expected)
	if len(tc.compatMemo) != 1 {
		t.Fatalf("expected memo size 1, got %d", len(tc.compatMemo))
	}
	if !tc.compatMemo[key] {
		t.Fatal("memo should record true")
	}
	// Force impl to panic if called again by corrupting memo hit path - just verify memo hit returns same
	if !tc.IsTypeCompatible(actual, expected) {
		t.Fatal("second call should hit memo")
	}
}
