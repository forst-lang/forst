package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestEnsureSuccessorNarrowing_blockBodySeesRefinedType(t *testing.T) {
	baseStr := ast.TypeString
	spanDeclX := ast.SourceSpan{StartLine: 1, StartCol: 3, EndLine: 1, EndCol: 4}
	spanEnsureX := ast.SourceSpan{StartLine: 1, StartCol: 30, EndLine: 1, EndCol: 31}
	spanBodyX := ast.SourceSpan{StartLine: 2, StartCol: 10, EndLine: 2, EndCol: 11}

	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "f"},
		// Functions that contain ensure are inferred as (T, Error).
		ReturnTypes: []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeError}},
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.VariableNode{{Ident: ast.Ident{ID: "x", Span: spanDeclX}}},
				RValues: []ast.ExpressionNode{ast.StringLiteralNode{Value: "hello"}},
				IsShort: true,
			},
			ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanEnsureX}},
				Assertion: ast.AssertionNode{
					BaseType: &baseStr,
				},
				Block: &ast.EnsureBlockNode{
					Body: []ast.Node{
						ast.ReturnNode{
							Values: []ast.ExpressionNode{
								ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanBodyX}},
								ast.NilLiteralNode{},
							},
						},
					},
				},
			},
			ast.ReturnNode{
				Values: []ast.ExpressionNode{
					ast.StringLiteralNode{Value: ""},
					ast.NilLiteralNode{},
				},
			},
		},
	}

	tc := New(logrus.New(), false)
	if err := tc.CheckTypes([]ast.Node{fn}); err != nil {
		t.Fatal(err)
	}

	ensureStmt := fn.Body[1].(ast.EnsureNode)
	ret := ensureStmt.Block.Body[0].(ast.ReturnNode)
	vn, ok := ret.Values[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("expected variable in return, got %T", ret.Values[0])
	}
	types, ok := tc.InferredTypesForVariableNode(vn)
	if !ok || len(types) != 1 {
		t.Fatalf("expected narrowed type for x in ensure block, got ok=%v len=%d", ok, len(types))
	}
	if types[0].Ident != ast.TypeString {
		t.Fatalf("expected String inside block, got %s", types[0].Ident)
	}
}

func TestEnsureSuccessorNarrowing_followingStatementsSeeRefinedTypeWithoutBlock(t *testing.T) {
	baseStr := ast.TypeString
	spanDeclX := ast.SourceSpan{StartLine: 1, StartCol: 3, EndLine: 1, EndCol: 4}
	// Match spans from TestEnsureSuccessorNarrowing_blockBodySeesRefinedType for the ensure subject.
	spanEnsureX := ast.SourceSpan{StartLine: 1, StartCol: 30, EndLine: 1, EndCol: 31}
	spanReturnX := ast.SourceSpan{StartLine: 1, StartCol: 40, EndLine: 1, EndCol: 41}

	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "f"},
		ReturnTypes: []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeError}},
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.VariableNode{{Ident: ast.Ident{ID: "x", Span: spanDeclX}}},
				RValues: []ast.ExpressionNode{ast.StringLiteralNode{Value: "hello"}},
				IsShort: true,
			},
			ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanEnsureX}},
				Assertion: ast.AssertionNode{
					BaseType: &baseStr,
				},
			},
			ast.ReturnNode{
				Values: []ast.ExpressionNode{
					ast.VariableNode{Ident: ast.Ident{ID: "x", Span: spanReturnX}},
					ast.NilLiteralNode{},
				},
			},
		},
	}

	tc := New(logrus.New(), false)
	if err := tc.CheckTypes([]ast.Node{fn}); err != nil {
		t.Fatal(err)
	}

	ret := fn.Body[2].(ast.ReturnNode)
	vn, ok := ret.Values[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("expected variable in return, got %T", ret.Values[0])
	}
	types, ok := tc.InferredTypesForVariableNode(vn)
	if !ok || len(types) != 1 {
		t.Fatalf("expected narrowed type for x after ensure, got ok=%v len=%d", ok, len(types))
	}
	if types[0].Ident != ast.TypeString {
		t.Fatalf("expected String after ensure, got %s", types[0].Ident)
	}
}
