package transformergo

import (
	"testing"

	"forst/internal/ast"
)

func TestGetEnsureBaseType_explicitBaseType(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	bt := ast.TypeString
	ensure := ast.EnsureNode{
		Assertion: ast.AssertionNode{BaseType: &bt},
	}
	got, err := tr.getEnsureBaseType(ensure)
	if err != nil {
		t.Fatal(err)
	}
	if got.Ident != ast.TypeString {
		t.Fatalf("got %v", got.Ident)
	}
}
