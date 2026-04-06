package typechecker

import (
	"io"
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func discardLogger() *logrus.Logger {
	log := logrus.New()
	log.SetOutput(io.Discard)
	return log
}

func TestApplyIfBranchNarrowing_nilConditionDoesNotPanic(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	tc.applyIfBranchNarrowing(nil)
}

func TestRefinedTypesForIsNarrowing_unsupportedRHSReturnsError(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)
	tc.CurrentScope().RegisterSymbol(ast.Identifier("x"), []ast.TypeNode{{Ident: ast.TypeString}}, SymbolVariable)

	_, err := tc.refinedTypesForIsNarrowing(
		ast.VariableNode{Ident: ast.Ident{ID: "x"}},
		ast.IntLiteralNode{Value: 1},
	)
	if err == nil || !strings.Contains(err.Error(), "unsupported RHS") {
		t.Fatalf("expected unsupported RHS error, got %v", err)
	}
}

func TestRefinedTypesForIsNarrowing_typeDefAssertionExprRHS(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)
	tc.CurrentScope().RegisterSymbol(ast.Identifier("x"), []ast.TypeNode{{Ident: ast.TypeString}}, SymbolVariable)

	str := ast.TypeString
	rhs := ast.TypeDefAssertionExpr{
		Assertion: &ast.AssertionNode{BaseType: &str},
	}
	types, err := tc.refinedTypesForIsNarrowing(ast.VariableNode{Ident: ast.Ident{ID: "x"}}, rhs)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || types[0].Ident != ast.TypeString {
		t.Fatalf("got %+v", types)
	}
}

func TestRefinedTypesForIsNarrowing_shapeNodeRHSEmptyShape(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)
	tc.CurrentScope().RegisterSymbol(ast.Identifier("x"), []ast.TypeNode{{Ident: ast.TypeString}}, SymbolVariable)

	types, err := tc.refinedTypesForIsNarrowing(
		ast.VariableNode{Ident: ast.Ident{ID: "x"}},
		ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || types[0].Ident == "" {
		t.Fatalf("expected one inferred shape type, got %+v", types)
	}
	if !strings.HasPrefix(string(types[0].Ident), "T_") {
		t.Fatalf("expected hash-backed empty shape type, got %s", types[0].Ident)
	}
}

func TestIfBranchNarrowing_elseIfBranchRefinesVariable(t *testing.T) {
	t.Parallel()
	baseStr := ast.TypeString
	spanDeclX := ast.SourceSpan{StartLine: 1, StartCol: 3, EndLine: 1, EndCol: 4}
	spanCondX := ast.SourceSpan{StartLine: 1, StartCol: 40, EndLine: 1, EndCol: 41}
	spanBodyX := ast.SourceSpan{StartLine: 1, StartCol: 55, EndLine: 1, EndCol: 56}

	fn := ast.FunctionNode{
		Ident:       ast.Ident{ID: "f"},
		ReturnTypes: []ast.TypeNode{{Ident: ast.TypeString}},
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanDeclX}}},
				RValues: []ast.ExpressionNode{ast.StringLiteralNode{Value: "hello"}},
				IsShort: true,
			},
			ast.IfNode{
				Condition: ast.BoolLiteralNode{Value: false},
				Body: []ast.Node{
					ast.ReturnNode{
						Values: []ast.ExpressionNode{ast.StringLiteralNode{Value: "first"}},
					},
				},
				ElseIfs: []ast.ElseIfNode{{
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
				}},
				Else: &ast.ElseBlockNode{
					Body: []ast.Node{
						ast.ReturnNode{
							Values: []ast.ExpressionNode{ast.StringLiteralNode{Value: ""}},
						},
					},
				},
			},
			ast.ReturnNode{
				Values: []ast.ExpressionNode{ast.StringLiteralNode{Value: ""}},
			},
		},
	}

	tc := New(discardLogger(), false)
	if err := tc.CheckTypes([]ast.Node{fn}); err != nil {
		t.Fatal(err)
	}

	ifn := fn.Body[1].(ast.IfNode)
	ei := ifn.ElseIfs[0]
	ret := ei.Body[0].(ast.ReturnNode)
	vn, ok := ret.Values[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("expected variable in else-if return, got %T", ret.Values[0])
	}
	types, ok := tc.InferredTypesForVariableNode(vn)
	if !ok || len(types) != 1 {
		t.Fatalf("expected narrowed type for x in else-if branch, got ok=%v len=%d", ok, len(types))
	}
	if types[0].Ident != ast.TypeString {
		t.Fatalf("expected String inside else-if branch, got %s", types[0].Ident)
	}
}

func TestApplyEnsureSuccessorNarrowing_skipsCompoundSubjectLeavesSymbolUnchanged(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)
	tc.CurrentScope().RegisterSymbol(ast.Identifier("a.b"), []ast.TypeNode{{Ident: ast.TypeString}}, SymbolVariable)

	str := ast.TypeString
	ensure := ast.EnsureNode{
		Variable: ast.VariableNode{Ident: ast.Ident{ID: "a.b"}},
		Assertion: ast.AssertionNode{
			BaseType: &str,
		},
	}
	tc.applyEnsureSuccessorNarrowing(ensure)

	sym, ok := tc.CurrentScope().LookupVariable(ast.Identifier("a.b"))
	if !ok || len(sym.Types) != 1 || sym.Types[0].Ident != ast.TypeString {
		t.Fatalf("compound subject must not be re-registered with refined types: ok=%v types=%+v", ok, sym.Types)
	}
}

func TestEndIfChainApplyJoin_emptyStackNoPanic(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	tc.endIfChainApplyJoin()
}

func TestIfChainNarrowing_beginRecordEnd_withoutBinding_mergeNoPanic(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	tc.beginIfChainForStatement()
	tc.recordIfChainNarrowingSubject(ast.Identifier("x"), []ast.TypeNode{{Ident: ast.TypeString}}, []string{"g"})
	tc.endIfChainApplyJoin()
}

func TestIfChainNarrowing_nestedChainsStackDiscipline(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	tc.beginIfChainForStatement()
	tc.beginIfChainForStatement()
	tc.recordIfChainNarrowingSubject(ast.Identifier("y"), []ast.TypeNode{{Ident: ast.TypeInt}}, nil)
	tc.endIfChainApplyJoin()
	tc.endIfChainApplyJoin()
}
