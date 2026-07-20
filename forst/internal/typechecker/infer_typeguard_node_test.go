package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestInferTypeGuardNode_rejectsReturnInBody(t *testing.T) {
	tc := New(setupTestLogger(nil), false)
	guard := &ast.TypeGuardNode{
		Ident: "BadReturn",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "x"},
			Type:  ast.TypeNode{Ident: ast.TypeInt},
		},
		Body: []ast.Node{
			ast.ReturnNode{Values: []ast.ExpressionNode{ast.BoolLiteralNode{Value: true}}},
		},
	}
	_, err := tc.inferTypeGuardNode(guard)
	if err == nil || !strings.Contains(err.Error(), "must not have return") {
		t.Fatalf("got %v", err)
	}
}

func TestInferTypeGuardNode_rejectsNonIsCondition(t *testing.T) {
	tc := New(setupTestLogger(nil), false)
	guard := &ast.TypeGuardNode{
		Ident: "BadCond",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "x"},
			Type:  ast.TypeNode{Ident: ast.TypeInt},
		},
		Body: []ast.Node{
			ast.IfNode{
				Condition: ast.BinaryExpressionNode{
					Left:     ast.VariableNode{Ident: ast.Ident{ID: "x"}},
					Operator: ast.TokenGreater,
					Right:    ast.IntLiteralNode{Value: 0},
				},
			},
		},
	}
	_, err := tc.inferTypeGuardNode(guard)
	if err == nil || !strings.Contains(err.Error(), "must use 'is'") {
		t.Fatalf("got %v", err)
	}
}

func TestInferTypeGuardNode_rejectsUnexpectedNode(t *testing.T) {
	tc := New(setupTestLogger(nil), false)
	guard := &ast.TypeGuardNode{
		Ident: "BadBody",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "x"},
			Type:  ast.TypeNode{Ident: ast.TypeInt},
		},
		Body: []ast.Node{
			ast.IntLiteralNode{Value: 1},
		},
	}
	_, err := tc.inferTypeGuardNode(guard)
	if err == nil || !strings.Contains(err.Error(), "may only contain if") {
		t.Fatalf("got %v", err)
	}
}

func TestInferTypeGuardNode_rejectsUnexpectedNodeType(t *testing.T) {
	tc := New(setupTestLogger(nil), false)
	_, err := tc.inferTypeGuardNode(ast.IntLiteralNode{Value: 1})
	if err == nil || !strings.Contains(err.Error(), "unexpected node type") {
		t.Fatalf("got %v", err)
	}
}
