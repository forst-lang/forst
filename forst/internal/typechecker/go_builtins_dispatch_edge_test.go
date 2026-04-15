package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestInferBuiltinArgType_missingIndex(t *testing.T) {
	tc := New(logrus.New(), false)
	_, err := tc.inferBuiltinArgType([]ast.ExpressionNode{}, 0, nil, ast.SourceSpan{})
	if err == nil {
		t.Fatal("expected missing argument error")
	}
	if !strings.Contains(err.Error(), "missing argument 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckBuiltinFunctionCall_dispatchKindWithoutHandlerReturnsInternalError(t *testing.T) {
	tc := New(logrus.New(), false)
	fn := BuiltinFunction{
		Name:      "unhandled-dispatch",
		Package:   "",
		CheckKind: BuiltinCheckDispatch,
	}

	_, err := tc.checkBuiltinFunctionCall(fn, nil, nil, ast.SourceSpan{})
	if err == nil {
		t.Fatal("expected internal missing dispatch error")
	}
	if !strings.Contains(err.Error(), "missing tryDispatchGoBuiltin case") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckBuiltinFunctionCall_genericArityMismatch(t *testing.T) {
	tc := New(logrus.New(), false)
	fn := BuiltinFunctions["os.Getenv"]

	_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
		ast.StringLiteralNode{Value: "A"},
		ast.StringLiteralNode{Value: "B"},
	}, nil, ast.SourceSpan{})
	if err == nil {
		t.Fatal("expected arity mismatch error for os.Getenv")
	}
	if !strings.Contains(err.Error(), "expects 1 arguments, got 2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckBuiltinFunctionCall_recoverRejectsArguments(t *testing.T) {
	tc := New(logrus.New(), false)
	fn := BuiltinFunctions["recover"]

	_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}}, nil, ast.SourceSpan{})
	if err == nil {
		t.Fatal("expected recover() arity error")
	}
	if !strings.Contains(err.Error(), "recover() expects 0 arguments, got 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckBuiltinFunctionCall_capRejectsString(t *testing.T) {
	tc := New(logrus.New(), false)
	fn := BuiltinFunctions["cap"]

	_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{ast.StringLiteralNode{Value: "abc"}}, nil, ast.SourceSpan{})
	if err == nil {
		t.Fatal("expected cap() invalid operand error")
	}
	if !strings.Contains(err.Error(), "cap() invalid operand type String") {
		t.Fatalf("unexpected error: %v", err)
	}
}
