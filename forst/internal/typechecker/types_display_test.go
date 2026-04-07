package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestFormatFunctionSignatureDisplay_ResultReturnIncludesError(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	sig := FunctionSignature{
		Ident: ast.Ident{ID: "f"},
		Parameters: []ParameterSignature{
			{Ident: ast.Ident{ID: "x"}, Type: ast.NewBuiltinType(ast.TypeString)},
		},
		ReturnTypes: []ast.TypeNode{
			ast.NewResultType(ast.NewBuiltinType(ast.TypeString), ast.NewBuiltinType(ast.TypeError)),
		},
	}
	got := tc.FormatFunctionSignatureDisplay(sig)
	want := "f(x String) -> Result(String, Error)"
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

func TestFormatVariableOccurrenceTypeForHover_narrowingGuardSuffix(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	sp := ast.SourceSpan{StartLine: 3, StartCol: 10, EndLine: 3, EndCol: 11}
	vn := ast.VariableNode{Ident: ast.Ident{ID: "x", Span: sp}}
	k := variableOccurrenceKey{ident: vn.Ident.ID, span: sp}
	tc.variableOccurrenceNarrowingGuards[k] = []string{"MyStr"}
	tc.variableOccurrenceNarrowingPredicateDisplay[k] = "MyStr()"
	got := tc.FormatVariableOccurrenceTypeForHover(vn, []ast.TypeNode{ast.NewBuiltinType(ast.TypeString)})
	if got != "String.MyStr()" {
		t.Fatalf("got %q want String.MyStr()", got)
	}
}

func TestPredicateChainForVariableHover_mergesNarrowingAndInferredTypeGuards(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	sp := ast.SourceSpan{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 2}
	vn := ast.VariableNode{Ident: ast.Ident{ID: "x", Span: sp}}
	k := variableOccurrenceKey{ident: vn.Ident.ID, span: sp}
	tc.variableOccurrenceNarrowingGuards[k] = []string{"Min"}
	tc.Defs[ast.TypeIdent("Strong")] = ast.TypeGuardNode{Ident: ast.Identifier("Strong")}
	pw := ast.TypeIdent("Password")
	tn := ast.TypeNode{
		Ident: pw,
		Assertion: &ast.AssertionNode{
			Constraints: []ast.ConstraintNode{{Name: "Strong"}},
		},
	}
	got := tc.PredicateChainForVariableHover(vn, []ast.TypeNode{tn})
	if len(got) != 2 || got[0] != "Min" || got[1] != "Strong" {
		t.Fatalf("got %v", got)
	}
}

func TestFormatVariableOccurrenceTypeForHover_noGuardsIsBaseOnly(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	sp := ast.SourceSpan{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 2}
	vn := ast.VariableNode{Ident: ast.Ident{ID: "x", Span: sp}}
	got := tc.FormatVariableOccurrenceTypeForHover(vn, []ast.TypeNode{ast.NewBuiltinType(ast.TypeInt)})
	if got != "Int" {
		t.Fatalf("got %q want Int", got)
	}
}
