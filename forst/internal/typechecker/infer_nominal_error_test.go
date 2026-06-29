package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func nominalErrorTC(t *testing.T) *TypeChecker {
	t.Helper()
	tc := New(logrus.New(), false)
	tc.registerType(ast.TypeDefNode{
		Ident: "NotFound",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"id": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	})
	return tc
}

func TestInferNominalErrorConstructorCall_validPayload(t *testing.T) {
	tc := nominalErrorTC(t)
	call := ast.FunctionCallNode{
		Function: ast.Ident{ID: "NotFound"},
		Arguments: []ast.ExpressionNode{
			ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"id": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}
	types, handled, err := tc.inferNominalErrorConstructorCall(call, nil)
	if err != nil || !handled {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if len(types) != 1 || types[0].Ident != "NotFound" {
		t.Fatalf("types = %#v", types)
	}
}

func TestInferNominalErrorConstructorCall_notErrorTypedef(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.registerType(ast.TypeDefNode{
		Ident: "Point",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	})
	call := ast.FunctionCallNode{
		Function: ast.Ident{ID: "Point"},
		Arguments: []ast.ExpressionNode{
			ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
				"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			}},
		},
	}
	_, handled, err := tc.inferNominalErrorConstructorCall(call, nil)
	if err != nil || handled {
		t.Fatalf("expected not handled for shape typedef, handled=%v err=%v", handled, err)
	}
}

func TestInferNominalErrorConstructorCall_wrongArity(t *testing.T) {
	tc := nominalErrorTC(t)
	call := ast.FunctionCallNode{
		Function:  ast.Ident{ID: "NotFound"},
		Arguments: []ast.ExpressionNode{},
	}
	_, handled, err := tc.inferNominalErrorConstructorCall(call, nil)
	if err == nil || !handled {
		t.Fatalf("expected arity diagnostic, handled=%v err=%v", handled, err)
	}
	diag, ok := err.(*Diagnostic)
	if !ok || diag.Code != "call-arity" {
		t.Fatalf("got %T %v", err, err)
	}
}

func TestInferNominalErrorConstructorCall_nonShapeArgument(t *testing.T) {
	tc := nominalErrorTC(t)
	call := ast.FunctionCallNode{
		Function:  ast.Ident{ID: "NotFound"},
		Arguments: []ast.ExpressionNode{ast.StringLiteralNode{Value: "x"}},
	}
	_, handled, err := tc.inferNominalErrorConstructorCall(call, nil)
	if err == nil || !handled {
		t.Fatalf("expected call-type diagnostic, handled=%v err=%v", handled, err)
	}
	diag, ok := err.(*Diagnostic)
	if !ok || diag.Code != "call-type" {
		t.Fatalf("got %T %v", err, err)
	}
}

func TestInferNominalErrorConstructorCall_payloadMismatch(t *testing.T) {
	tc := nominalErrorTC(t)
	call := ast.FunctionCallNode{
		Function: ast.Ident{ID: "NotFound"},
		Arguments: []ast.ExpressionNode{
			ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"wrong": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	}
	_, handled, err := tc.inferNominalErrorConstructorCall(call, nil)
	if err == nil || !handled {
		t.Fatalf("expected payload mismatch, handled=%v err=%v", handled, err)
	}
	if !strings.Contains(err.Error(), "payload does not match") {
		t.Fatalf("err = %v", err)
	}
}

func TestInferNominalErrorConstructorCall_unknownFunction(t *testing.T) {
	tc := New(logrus.New(), false)
	call := ast.FunctionCallNode{
		Function:  ast.Ident{ID: "Missing"},
		Arguments: []ast.ExpressionNode{ast.ShapeNode{}},
	}
	_, handled, err := tc.inferNominalErrorConstructorCall(call, nil)
	if err != nil || handled {
		t.Fatalf("expected not handled for unknown fn, handled=%v err=%v", handled, err)
	}
}
