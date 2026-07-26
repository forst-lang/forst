package gointerop_test

import (
	"go/types"
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker/gointerop"
)

func TestTypeToForstType_mapsErrorInterface(t *testing.T) {
	t.Parallel()
	errIface := gointerop.ErrorInterfaceType()
	if errIface == nil {
		t.Fatal("expected error interface")
	}
	got, ok := gointerop.TypeToForstType(errIface)
	if !ok || got.Ident != ast.TypeError {
		t.Fatalf("want Error, got %v ok=%v", got, ok)
	}
}

func TestIdentifierExported(t *testing.T) {
	t.Parallel()
	if !gointerop.IdentifierExported("Foo") {
		t.Fatal("Foo should be exported")
	}
	if gointerop.IdentifierExported("foo") {
		t.Fatal("foo should not be exported")
	}
}

func TestCheckSignature_multiReturnMapsToTuple(t *testing.T) {
	t.Parallel()
	r0 := types.NewVar(0, nil, "", types.Typ[types.Int])
	r1 := types.NewVar(0, nil, "", types.Universe.Lookup("error").Type())
	results := types.NewTuple(r0, r1)
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), results, false)

	host := stubHost{}
	diag := func(span ast.SourceSpan, code, format string, args ...any) error {
		return nil
	}
	got, err := gointerop.CheckSignature(host, diag, gointerop.SignatureCheck{
		Sig:             sig,
		Qual:            "test.F",
		WantSingleValue: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].IsTupleType() {
		t.Fatalf("want Tuple, got %v", got)
	}
}

type stubHost struct{}

func (stubHost) ForstTypeForGoType(g types.Type) (ast.TypeNode, bool) {
	return ast.TypeNode{}, false
}

func (stubHost) IsTypeCompatible(a, b ast.TypeNode) bool {
	return false
}

func (stubHost) InferExpressionType(expr ast.ExpressionNode) ([]ast.TypeNode, error) {
	return nil, nil
}
