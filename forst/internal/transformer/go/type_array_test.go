package transformergo

import (
	"testing"

	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

func TestTransformType_fixedArray(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	out, err := tr.transformType(ast.NewFixedArrayType(ast.NewBuiltinType(ast.TypeInt), 3))
	if err != nil {
		t.Fatal(err)
	}
	arr, ok := out.(*goast.ArrayType)
	if !ok || arr.Len == nil {
		t.Fatalf("got %#v", out)
	}
	if lit, ok := arr.Len.(*goast.BasicLit); !ok || lit.Value != "3" {
		t.Fatalf("len = %#v", arr.Len)
	}
}

func TestTransformType_sliceOmitsLen(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	out, err := tr.transformType(ast.NewArrayType(ast.NewBuiltinType(ast.TypeInt)))
	if err != nil {
		t.Fatal(err)
	}
	arr, ok := out.(*goast.ArrayType)
	if !ok || arr.Len != nil {
		t.Fatalf("got %#v", out)
	}
	if _, ok := arr.Elt.(*goast.Ident); !ok {
		t.Fatalf("elt = %#v", arr.Elt)
	}
	_ = token.INT
}
