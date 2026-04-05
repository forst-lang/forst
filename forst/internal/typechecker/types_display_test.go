package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestFormatFunctionSignatureDisplay_TupleReturnIncludesError(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	sig := FunctionSignature{
		Ident: ast.Ident{ID: "f"},
		Parameters: []ParameterSignature{
			{Ident: ast.Ident{ID: "x"}, Type: ast.NewBuiltinType(ast.TypeString)},
		},
		ReturnTypes: []ast.TypeNode{
			ast.NewBuiltinType(ast.TypeString),
			ast.NewBuiltinType(ast.TypeError),
		},
	}
	got := tc.FormatFunctionSignatureDisplay(sig)
	want := "f(x String) -> (String, Error)"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatFunctionSignatureDisplay_SingleReturnUnparenthesized(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	sig := FunctionSignature{
		Ident:       ast.Ident{ID: "g"},
		Parameters:  nil,
		ReturnTypes: []ast.TypeNode{ast.NewBuiltinType(ast.TypeVoid)},
	}
	got := tc.FormatFunctionSignatureDisplay(sig)
	want := "g() -> Void"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatTypeNodeDisplay_Builtin(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	got := tc.FormatTypeNodeDisplay(ast.NewBuiltinType(ast.TypeError))
	if got != "Error" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatTypeNodeDisplay_mapsAssertionRefinementHashToTypeAlias(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	myStr := ast.TypeIdent("MyStr")
	str := ast.TypeString
	tc.Defs[myStr] = ast.TypeDefNode{
		Ident: myStr,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &str,
			},
		},
	}
	a := ast.AssertionNode{BaseType: &myStr}
	h, err := tc.Hasher.HashNode(a)
	if err != nil {
		t.Fatal(err)
	}
	hashIdent := h.ToTypeIdent()
	tc.Defs[hashIdent] = ast.TypeDefNode{
		Ident: hashIdent,
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}},
		},
	}
	got := tc.FormatTypeNodeDisplay(ast.TypeNode{Ident: hashIdent})
	if got != "MyStr" {
		t.Fatalf("got %q want MyStr", got)
	}
}

func TestInferredTypesForVariableIdentifier_FromTypesMap(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	v := ast.VariableNode{Ident: ast.Ident{ID: "n"}}
	tc.storeInferredType(v, []ast.TypeNode{ast.NewBuiltinType(ast.TypeInt)})
	got, ok := tc.InferredTypesForVariableIdentifier("n")
	if !ok || len(got) != 1 || got[0].Ident != ast.TypeInt {
		t.Fatalf("got %v ok=%v", got, ok)
	}
}

func TestInferredTypesForVariableIdentifier_FromVariableTypesFallback(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	tc.VariableTypes["z"] = []ast.TypeNode{ast.NewBuiltinType(ast.TypeString)}
	got, ok := tc.InferredTypesForVariableIdentifier("z")
	if !ok || len(got) != 1 || got[0].Ident != ast.TypeString {
		t.Fatalf("got %v ok=%v", got, ok)
	}
}
