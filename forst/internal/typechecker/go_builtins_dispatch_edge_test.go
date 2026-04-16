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

func TestCheckBuiltinFunctionCall_dispatchArityAndOperandErrors(t *testing.T) {
	tc := New(logrus.New(), false)
	span := ast.SourceSpan{}

	t.Run("len_wrong_arity", func(t *testing.T) {
		fn := BuiltinFunctions["len"]
		_, err := tc.checkBuiltinFunctionCall(fn, nil, nil, span)
		if err == nil || !strings.Contains(err.Error(), "len() expects 1 argument") {
			t.Fatalf("got %v", err)
		}
		_, err = tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.StringLiteralNode{Value: "a"},
			ast.StringLiteralNode{Value: "b"},
		}, nil, span)
		if err == nil || !strings.Contains(err.Error(), "len() expects 1 argument") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("append_not_enough_args", func(t *testing.T) {
		fn := BuiltinFunctions["append"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.ArrayLiteralNode{Value: []ast.LiteralNode{ast.IntLiteralNode{Value: 1}}},
		}, nil, span)
		if err == nil || !strings.Contains(err.Error(), "append() expects at least 2 arguments") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("append_first_not_slice", func(t *testing.T) {
		fn := BuiltinFunctions["append"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.IntLiteralNode{Value: 1},
			ast.IntLiteralNode{Value: 2},
		}, nil, span)
		if err == nil || !strings.Contains(err.Error(), "append() first argument must be a slice") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("append_elem_mismatch", func(t *testing.T) {
		fn := BuiltinFunctions["append"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.ArrayLiteralNode{Value: []ast.LiteralNode{ast.IntLiteralNode{Value: 1}}},
			ast.StringLiteralNode{Value: "x"},
		}, nil, span)
		if err == nil || !strings.Contains(err.Error(), "must be assignable to slice element") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("copy_wrong_arity", func(t *testing.T) {
		fn := BuiltinFunctions["copy"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.ArrayLiteralNode{Value: []ast.LiteralNode{ast.IntLiteralNode{Value: 1}}},
		}, nil, span)
		if err == nil || !strings.Contains(err.Error(), "copy() expects 2 arguments") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("copy_mismatched_slice_elems", func(t *testing.T) {
		fn := BuiltinFunctions["copy"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.ArrayLiteralNode{Value: []ast.LiteralNode{ast.IntLiteralNode{Value: 1}}},
			ast.ArrayLiteralNode{Value: []ast.LiteralNode{ast.StringLiteralNode{Value: "a"}}},
		}, nil, span)
		if err == nil || !strings.Contains(err.Error(), "element types must match") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("delete_wrong_arity", func(t *testing.T) {
		fn := BuiltinFunctions["delete"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.StringLiteralNode{Value: "m"},
		}, nil, span)
		if err == nil || !strings.Contains(err.Error(), "delete() expects 2 arguments") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("clear_invalid_operand", func(t *testing.T) {
		fn := BuiltinFunctions["clear"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.IntLiteralNode{Value: 1},
		}, nil, span)
		if err == nil || !strings.Contains(err.Error(), "clear() expects a map or slice") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("close_wrong_arity", func(t *testing.T) {
		fn := BuiltinFunctions["close"]
		_, err := tc.checkBuiltinFunctionCall(fn, nil, nil, span)
		if err == nil || !strings.Contains(err.Error(), "close() expects 1 argument") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("min_zero_args", func(t *testing.T) {
		fn := BuiltinFunctions["min"]
		_, err := tc.checkBuiltinFunctionCall(fn, nil, nil, span)
		if err == nil || !strings.Contains(err.Error(), "expects at least 1 argument") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("min_first_not_ordered", func(t *testing.T) {
		fn := BuiltinFunctions["min"]
		complexCall := ast.FunctionCallNode{
			Function: ast.Ident{ID: "complex"},
			Arguments: []ast.ExpressionNode{
				ast.FloatLiteralNode{Value: 1},
				ast.FloatLiteralNode{Value: 0},
			},
		}
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{complexCall}, nil, span)
		if err == nil || !strings.Contains(err.Error(), "ordered types") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("complex_wrong_arity", func(t *testing.T) {
		fn := BuiltinFunctions["complex"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.FloatLiteralNode{Value: 1},
		}, nil, span)
		if err == nil || !strings.Contains(err.Error(), "complex() expects 2 arguments") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("complex_non_float", func(t *testing.T) {
		fn := BuiltinFunctions["complex"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.IntLiteralNode{Value: 1},
			ast.FloatLiteralNode{Value: 0},
		}, nil, span)
		if err == nil || !strings.Contains(err.Error(), "complex() expects Float arguments") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("real_not_complex", func(t *testing.T) {
		fn := BuiltinFunctions["real"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.IntLiteralNode{Value: 1},
		}, nil, span)
		if err == nil || !strings.Contains(err.Error(), "expects a complex value") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("panic_wrong_arity", func(t *testing.T) {
		fn := BuiltinFunctions["panic"]
		_, err := tc.checkBuiltinFunctionCall(fn, nil, nil, span)
		if err == nil || !strings.Contains(err.Error(), "panic() expects 1 argument") {
			t.Fatalf("got %v", err)
		}
	})
}
