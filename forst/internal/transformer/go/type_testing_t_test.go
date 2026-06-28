package transformergo

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestTransformType_pointerTestingT(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	ptr := ast.TypeNode{
		Ident:      ast.TypePointer,
		TypeParams: []ast.TypeNode{{Ident: "testing.T"}},
	}
	expr, err := tr.transformType(ptr)
	if err != nil {
		t.Fatal(err)
	}
	s := goExprString(t, expr)
	if !strings.Contains(s, "*testing.T") {
		t.Fatalf("got %q", s)
	}
}
