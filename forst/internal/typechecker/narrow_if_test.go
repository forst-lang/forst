package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestIfBranchNarrowing_typeAliasIsMyStrRefinesSubjectToString(t *testing.T) {
	t.Parallel()
	myStr := ast.TypeIdent("MyStr")
	str := ast.TypeString
	typeDef := ast.TypeDefNode{
		Ident: myStr,
		Expr: &ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &str,
			},
		},
	}

	spanDeclX := ast.SourceSpan{StartLine: 1, StartCol: 3, EndLine: 1, EndCol: 4}
	spanCondX := ast.SourceSpan{StartLine: 1, StartCol: 22, EndLine: 1, EndCol: 23}
	spanBodyX := ast.SourceSpan{StartLine: 1, StartCol: 35, EndLine: 1, EndCol: 36}

	fn := ast.FunctionNode{
		Ident:       ast.Ident{ID: "f"},
		ReturnTypes: []ast.TypeNode{{Ident: ast.TypeString}},
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.VariableNode{{Ident: ast.Ident{ID: "x", Span: spanDeclX}}},
				RValues: []ast.ExpressionNode{ast.StringLiteralNode{Value: "hello"}},
				IsShort: true,
			},
			ast.IfNode{
				Condition: ast.BinaryExpressionNode{
					Left:     ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanCondX}},
					Operator: ast.TokenIs,
					Right: ast.AssertionNode{
						BaseType: func() *ast.TypeIdent {
							id := myStr
							return &id
						}(),
					},
				},
				Body: []ast.Node{
					ast.ReturnNode{
						Values: []ast.ExpressionNode{
							ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanBodyX}},
						},
					},
				},
			},
			ast.ReturnNode{
				Values: []ast.ExpressionNode{ast.StringLiteralNode{Value: ""}},
			},
		},
	}

	tc := New(logrus.New(), false)
	if err := tc.CheckTypes([]ast.Node{typeDef, fn}); err != nil {
		t.Fatal(err)
	}

	ifn := fn.Body[1].(ast.IfNode)
	ret := ifn.Body[0].(ast.ReturnNode)
	vn, ok := ret.Values[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("expected variable in return, got %T", ret.Values[0])
	}
	types, ok := tc.InferredTypesForVariableNode(vn)
	if !ok || len(types) != 1 {
		t.Fatalf("expected refined type for x in then-branch, got ok=%v len=%d", ok, len(types))
	}
	if types[0].Ident != ast.TypeString {
		t.Fatalf("expected String (underlying of MyStr) inside branch, got %s", types[0].Ident)
	}
}

func TestIfBranchNarrowing_thenBranchVariableGetsRefinedType(t *testing.T) {
	baseStr := ast.TypeString
	// Distinct spans so VariableNode hashes differ (mirrors parser behavior).
	spanDeclX := ast.SourceSpan{StartLine: 1, StartCol: 3, EndLine: 1, EndCol: 4}
	spanCondX := ast.SourceSpan{StartLine: 1, StartCol: 22, EndLine: 1, EndCol: 23}
	spanBodyX := ast.SourceSpan{StartLine: 1, StartCol: 35, EndLine: 1, EndCol: 36}

	fn := ast.FunctionNode{
		Ident:       ast.Ident{ID: "f"},
		ReturnTypes: []ast.TypeNode{{Ident: ast.TypeString}},
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.VariableNode{{Ident: ast.Ident{ID: "x", Span: spanDeclX}}},
				RValues: []ast.ExpressionNode{ast.StringLiteralNode{Value: "hello"}},
				IsShort: true,
			},
			ast.IfNode{
				Condition: ast.BinaryExpressionNode{
					Left:     ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanCondX}},
					Operator: ast.TokenIs,
					Right: ast.AssertionNode{
						BaseType: &baseStr,
					},
				},
				Body: []ast.Node{
					ast.ReturnNode{
						Values: []ast.ExpressionNode{
							ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanBodyX}},
						},
					},
				},
			},
			ast.ReturnNode{
				Values: []ast.ExpressionNode{ast.StringLiteralNode{Value: ""}},
			},
		},
	}

	tc := New(logrus.New(), false)
	if err := tc.CheckTypes([]ast.Node{fn}); err != nil {
		t.Fatal(err)
	}

	ifn := fn.Body[1].(ast.IfNode)
	ret := ifn.Body[0].(ast.ReturnNode)
	vn, ok := ret.Values[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("expected variable in return, got %T", ret.Values[0])
	}
	types, ok := tc.InferredTypesForVariableNode(vn)
	if !ok || len(types) != 1 {
		t.Fatalf("expected narrowed type for x in then-branch, got ok=%v len=%d", ok, len(types))
	}
	if types[0].Ident != ast.TypeString {
		t.Fatalf("expected String inside branch, got %s", types[0].Ident)
	}
}

func TestRefinedNarrowingTypeFromAliasAssertion_baseTypeNamesTypeGuardUsesSubjectType(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	tc.Defs[ast.TypeIdent("MyStr")] = ast.TypeGuardNode{
		Ident: "MyStr",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "x"},
			Type:  ast.TypeNode{Ident: ast.TypeString},
		},
		Body: []ast.Node{},
	}
	base := ast.TypeIdent("MyStr")
	a := &ast.AssertionNode{BaseType: &base}
	refined := []ast.TypeNode{{Ident: ast.TypeIdent("T_shouldNotAppear")}}
	got := tc.refinedNarrowingTypeFromAliasAssertion(a, refined)
	if len(got) != 1 || got[0].Ident != ast.TypeString {
		t.Fatalf("want narrowed to guard subject String, got %+v", got)
	}
}

func TestUnderlyingBuiltinTypeOfAliasAssertion_aliasOverBuiltin(t *testing.T) {
	t.Parallel()
	myStr := ast.TypeIdent("MyStr")
	str := ast.TypeString
	typeDef := ast.TypeDefNode{
		Ident: myStr,
		Expr: &ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &str,
			},
		},
	}
	tc := New(logrus.New(), false)
	if err := tc.CheckTypes([]ast.Node{typeDef}); err != nil {
		t.Fatal(err)
	}
	if got := tc.underlyingBuiltinTypeOfAliasAssertion(myStr); got != ast.TypeString {
		t.Fatalf("expected String, got %q", got)
	}
}

func TestUnderlyingBuiltinTypeOfAliasAssertion_nestedAliasChain(t *testing.T) {
	t.Parallel()
	myStr := ast.TypeIdent("MyStr")
	inner := ast.TypeIdent("Inner")
	str := ast.TypeString
	nodes := []ast.Node{
		ast.TypeDefNode{
			Ident: inner,
			Expr: &ast.TypeDefAssertionExpr{
				Assertion: &ast.AssertionNode{BaseType: &str},
			},
		},
		ast.TypeDefNode{
			Ident: myStr,
			Expr: &ast.TypeDefAssertionExpr{
				Assertion: &ast.AssertionNode{BaseType: &inner},
			},
		},
	}
	tc := New(logrus.New(), false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	if got := tc.underlyingBuiltinTypeOfAliasAssertion(myStr); got != ast.TypeString {
		t.Fatalf("expected String, got %q", got)
	}
}

func TestVariableOccurrenceTypes_storesDistinctTypesPerSpan(t *testing.T) {
	tc := New(logrus.New(), false)
	sa := ast.SourceSpan{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 2}
	sb := ast.SourceSpan{StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 2}
	va := ast.VariableNode{Ident: ast.Ident{ID: "x", Span: sa}}
	vb := ast.VariableNode{Ident: ast.Ident{ID: "x", Span: sb}}
	tc.storeInferredType(va, []ast.TypeNode{{Ident: ast.TypeString}})
	tc.storeInferredType(vb, []ast.TypeNode{{Ident: ast.TypeInt}})
	ta, ok := tc.InferredTypesForVariableNode(va)
	if !ok || len(ta) != 1 || ta[0].Ident != ast.TypeString {
		t.Fatalf("first occurrence: got %+v ok=%v", ta, ok)
	}
	tb, ok := tc.InferredTypesForVariableNode(vb)
	if !ok || len(tb) != 1 || tb[0].Ident != ast.TypeInt {
		t.Fatalf("second occurrence: got %+v ok=%v", tb, ok)
	}
}
