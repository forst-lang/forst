package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestSpanOfExpression_functionCallPrefersCallSpan(t *testing.T) {
	t.Parallel()
	call := ast.FunctionCallNode{
		CallSpan: ast.SourceSpan{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 5},
		Function: ast.Ident{ID: "f"},
	}
	if s := spanOfExpression(call); !s.IsSet() || s.StartCol != 1 {
		t.Fatalf("got %+v", s)
	}
}

func TestSpanOfExpression_functionCallFallsBackToFunctionSpan(t *testing.T) {
	t.Parallel()
	call := ast.FunctionCallNode{
		Function: ast.Ident{
			ID:   "g",
			Span: ast.SourceSpan{StartLine: 2, StartCol: 3, EndLine: 2, EndCol: 4},
		},
	}
	if s := spanOfExpression(call); !s.IsSet() || s.StartLine != 2 {
		t.Fatalf("got %+v", s)
	}
}

func TestSpanOfExpression_variableUsesIdentSpan(t *testing.T) {
	t.Parallel()
	v := ast.VariableNode{Ident: ast.Ident{ID: "x", Span: ast.SourceSpan{StartLine: 5, StartCol: 2, EndLine: 5, EndCol: 3}}}
	if s := spanOfExpression(v); !s.IsSet() || s.StartLine != 5 {
		t.Fatalf("got %+v", s)
	}
}

func TestSpanForCallArg_preferArgSpan(t *testing.T) {
	t.Parallel()
	argSpan := ast.SourceSpan{StartLine: 1, StartCol: 10, EndLine: 1, EndCol: 11}
	args := []ast.ExpressionNode{ast.IntLiteralNode{Value: 1, Span: ast.FakeSpan()}}
	got := spanForCallArg([]ast.SourceSpan{argSpan}, 0, args, ast.SourceSpan{})
	if !got.IsSet() || got.StartCol != 10 {
		t.Fatalf("got %+v", got)
	}
}

func TestSpanOfExpression_intLiteralUsesSpan(t *testing.T) {
	t.Parallel()
	lit := ast.IntLiteralNode{Value: 42, Span: ast.SourceSpan{StartLine: 4, StartCol: 2, EndLine: 4, EndCol: 4}}
	if s := spanOfExpression(lit); !s.IsSet() || s.StartLine != 4 {
		t.Fatalf("got %+v", s)
	}
}

func TestSpanOfExpression_indexFallsBackToIndexLiteral(t *testing.T) {
	t.Parallel()
	idx := ast.IndexExpressionNode{
		Target: ast.VariableNode{Ident: ast.Ident{ID: "a"}},
		Index:  ast.IntLiteralNode{Value: 9, Span: ast.SourceSpan{StartLine: 7, StartCol: 5, EndLine: 7, EndCol: 6}},
	}
	if s := spanOfExpression(idx); !s.IsSet() || s.StartLine != 7 {
		t.Fatalf("got %+v", s)
	}
}

func TestSpanForCallArg_fallbackToCallSpan(t *testing.T) {
	t.Parallel()
	callSpan := ast.SourceSpan{StartLine: 3, StartCol: 1, EndLine: 3, EndCol: 9}
	got := spanForCallArg(nil, 0, nil, callSpan)
	if !got.IsSet() || got.StartLine != 3 {
		t.Fatalf("got %+v", got)
	}
}
