package transformergo

import (
	"go/ast"
	"io"
	"testing"

	forstast "forst/internal/ast"
	"forst/internal/typechecker"
)

// Exercises transformType's TypeUnion branch: error-kinded unions lower to Go's built-in error type;
// mixed unions lower to any (same path as typedef fallback for non-sealed unions).
func TestTransformType_unionErrorKindedVsAny(t *testing.T) {
	t.Parallel()
	log := forstast.SetupTestLogger(nil)
	log.SetOutput(io.Discard)
	tc := typechecker.New(log, false)
	tc.Defs["E1"] = forstast.TypeDefNode{
		Ident: "E1",
		Expr: forstast.TypeDefErrorExpr{
			Payload: forstast.ShapeNode{Fields: map[string]forstast.ShapeFieldNode{}},
		},
	}
	tc.Defs["E2"] = forstast.TypeDefNode{
		Ident: "E2",
		Expr: forstast.TypeDefErrorExpr{
			Payload: forstast.ShapeNode{Fields: map[string]forstast.ShapeFieldNode{}},
		},
	}
	tr := New(tc, log)

	errKinded := forstast.NewUnionType(
		forstast.TypeNode{Ident: "E1"},
		forstast.TypeNode{Ident: "E2"},
	)
	goErr, err := tr.transformType(errKinded)
	if err != nil {
		t.Fatal(err)
	}
	id, ok := goErr.(*ast.Ident)
	if !ok || id.Name != "error" {
		t.Fatalf("nominal error union should emit Go `error`, got %#v", goErr)
	}

	mixed := forstast.NewUnionType(
		forstast.TypeNode{Ident: forstast.TypeError},
		forstast.TypeNode{Ident: forstast.TypeString},
	)
	goAny, err := tr.transformType(mixed)
	if err != nil {
		t.Fatal(err)
	}
	id2, ok := goAny.(*ast.Ident)
	if !ok || id2.Name != "any" {
		t.Fatalf("non-uniformly error-kinded union should emit `any`, got %#v", goAny)
	}

	inter := forstast.NewIntersectionType(
		forstast.TypeNode{Ident: "E1"},
		forstast.TypeNode{Ident: "E2"},
	)
	goInter, err := tr.transformType(inter)
	if err != nil {
		t.Fatal(err)
	}
	id3, ok := goInter.(*ast.Ident)
	if !ok || id3.Name != "any" {
		t.Fatalf("intersection should lower to `any` in Go, got %#v", goInter)
	}
}
