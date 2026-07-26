package transformergo

import (
	goast "go/ast"
	"testing"

	forstast "forst/internal/ast"
	"forst/internal/typechecker"
)

func TestTransformFunction_variadicParameterEmitsGoVariadic(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(nil, false)
	tc.Functions[forstast.Identifier("sum")] = typechecker.FunctionSignature{
		Parameters: []typechecker.ParameterSignature{
			{Ident: forstast.Ident{ID: "nums"}, Type: forstast.TypeNode{Ident: forstast.TypeInt}, Variadic: true},
		},
		ReturnTypes: []forstast.TypeNode{{Ident: forstast.TypeInt}},
	}
	tr := setupTransformer(tc, nil)
	params := []forstast.ParamNode{
		forstast.SimpleParamNode{
			Ident:    forstast.Ident{ID: "nums"},
			Type:     forstast.TypeNode{Ident: forstast.TypeInt},
			Variadic: true,
		},
	}
	fields, err := tr.transformFunctionParams(forstast.Identifier("sum"), params)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields.List) != 1 {
		t.Fatalf("want 1 param, got %d", len(fields.List))
	}
	if _, ok := fields.List[0].Type.(*goast.Ellipsis); !ok {
		t.Fatalf("want ...int param, got %#v", fields.List[0].Type)
	}
	ell, ok := fields.List[0].Type.(*goast.Ellipsis)
	if !ok {
		t.Fatal("expected ellipsis type")
	}
	ident, ok := ell.Elt.(*goast.Ident)
	if !ok || ident.Name != "int" {
		t.Fatalf("want int elem, got %#v", ell.Elt)
	}
}
