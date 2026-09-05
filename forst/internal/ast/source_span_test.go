package ast

import "testing"

func TestSourceSpan_ContainsPosition(t *testing.T) {
	t.Parallel()
	s := SourceSpan{StartLine: 2, StartCol: 3, EndLine: 2, EndCol: 6} // covers "abc"
	if !s.ContainsPosition(2, 3) {
		t.Fatal("start inclusive")
	}
	if s.ContainsPosition(2, 6) {
		t.Fatal("end exclusive")
	}
	mid := SourceSpan{StartLine: 1, StartCol: 1, EndLine: 3, EndCol: 1}
	if !mid.ContainsPosition(2, 5) {
		t.Fatal("middle line")
	}
}

func TestSourceSpan_ContainsPosition_unsetSpan(t *testing.T) {
	t.Parallel()
	if (SourceSpan{}).ContainsPosition(1, 1) {
		t.Fatal("unset span should contain nothing")
	}
}

func TestSourceSpan_ContainsPosition_outsideLineRange(t *testing.T) {
	t.Parallel()
	s := SourceSpan{StartLine: 2, StartCol: 1, EndLine: 4, EndCol: 5}
	if s.ContainsPosition(1, 1) {
		t.Fatal("before start line")
	}
	if s.ContainsPosition(5, 1) {
		t.Fatal("after end line")
	}
}

func TestSourceSpan_ContainsPosition_multilineFirstAndLastLineEdges(t *testing.T) {
	t.Parallel()
	// Lines 1–3; end column on line 3 is exclusive.
	s := SourceSpan{StartLine: 1, StartCol: 3, EndLine: 3, EndCol: 2}
	if s.ContainsPosition(1, 2) {
		t.Fatal("on first line but before StartCol")
	}
	if !s.ContainsPosition(1, 3) {
		t.Fatal("on first line at StartCol")
	}
	if !s.ContainsPosition(3, 1) {
		t.Fatal("on last line before EndCol")
	}
	if s.ContainsPosition(3, 2) {
		t.Fatal("on last line at EndCol (exclusive)")
	}
}

func TestSourceSpan_IsSet(t *testing.T) {
	t.Parallel()
	if (SourceSpan{}).IsSet() {
		t.Fatal("zero span should be unset")
	}
	if !(SourceSpan{StartLine: 1, StartCol: 1}).IsSet() {
		t.Fatal("1,1 should be set")
	}
	if (SourceSpan{StartLine: 0, StartCol: 1}).IsSet() {
		t.Fatal("line 0 should be unset")
	}
}

func TestSpanFromToken(t *testing.T) {
	t.Parallel()
	s := SpanFromToken(Token{Line: 2, Column: 3, Value: "ab"})
	if s.StartLine != 2 || s.StartCol != 3 || s.EndLine != 2 || s.EndCol != 5 {
		t.Fatalf("got %+v", s)
	}
}

func TestSpanFromToken_emptyValueUsesMinWidth(t *testing.T) {
	t.Parallel()
	s := SpanFromToken(Token{Line: 1, Column: 4, Value: ""})
	if s.EndCol != 5 {
		t.Fatalf("EndCol = %d want 5", s.EndCol)
	}
}

func TestSpanBetweenTokens(t *testing.T) {
	t.Parallel()
	start := Token{Line: 1, Column: 1, Value: "for"}
	end := Token{Line: 3, Column: 10, Value: "done"}
	s := SpanBetweenTokens(start, end)
	if s.StartLine != 1 || s.StartCol != 1 || s.EndLine != 3 || s.EndCol != 14 {
		t.Fatalf("got %+v", s)
	}
}

func TestSpanBetweenTokens_emptyEndValueUsesMinWidth(t *testing.T) {
	t.Parallel()
	s := SpanBetweenTokens(
		Token{Line: 1, Column: 1, Value: "a"},
		Token{Line: 1, Column: 5, Value: ""},
	)
	if s.EndCol != 6 {
		t.Fatalf("EndCol = %d want 6", s.EndCol)
	}
}

func TestFakeSpan_isSet(t *testing.T) {
	t.Parallel()
	if !FakeSpan().IsSet() {
		t.Fatal("FakeSpan must be set")
	}
}

func TestSpanFromTo_mergesEnds(t *testing.T) {
	t.Parallel()
	start := SourceSpan{StartLine: 1, StartCol: 2, EndLine: 1, EndCol: 3}
	end := SourceSpan{StartLine: 1, StartCol: 8, EndLine: 1, EndCol: 9}
	got := SpanFromTo(start, end)
	if got.StartLine != 1 || got.StartCol != 2 || got.EndLine != 1 || got.EndCol != 9 {
		t.Fatalf("got %+v", got)
	}
}

func TestExpressionSpanStart_variableAndIndex(t *testing.T) {
	t.Parallel()
	v := VariableNode{Ident: Ident{ID: "xs", Span: SourceSpan{StartLine: 1, StartCol: 5, EndLine: 1, EndCol: 7}}}
	idx := IndexExpressionNode{
		Target: v,
		Index:  IntLiteralNode{Value: 1, Span: FakeSpan()},
		Span:   SourceSpan{StartLine: 1, StartCol: 5, EndLine: 1, EndCol: 10},
	}
	cases := []struct {
		name string
		expr ExpressionNode
		want SourceSpan
	}{
		{name: "variable", expr: v, want: v.Ident.Span},
		{name: "index", expr: idx, want: idx.Span},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExpressionSpanStart(tc.expr)
			if got != tc.want {
				t.Fatalf("got %+v want %+v", got, tc.want)
			}
		})
	}
}

func TestExpressionSpanStart_callStartsAtCalleeNotCallSpan(t *testing.T) {
	t.Parallel()
	fnSpan := SourceSpan{StartLine: 1, StartCol: 2, EndLine: 1, EndCol: 3}
	callSpan := SourceSpan{StartLine: 1, StartCol: 3, EndLine: 1, EndCol: 5}
	call := FunctionCallNode{
		Function: Ident{ID: "f", Span: fnSpan},
		CallSpan: callSpan,
	}
	recvSpan := SourceSpan{StartLine: 2, StartCol: 4, EndLine: 2, EndCol: 7}
	methodCall := MethodCallNode{
		Receiver: VariableNode{Ident: Ident{ID: "recv", Span: recvSpan}},
		Method:   Ident{ID: "m", Span: SourceSpan{StartLine: 2, StartCol: 8, EndLine: 2, EndCol: 9}},
		CallSpan: SourceSpan{StartLine: 2, StartCol: 9, EndLine: 2, EndCol: 11},
	}
	okSpan := SourceSpan{StartLine: 3, StartCol: 1, EndLine: 3, EndCol: 7}
	ok := OkExprNode{
		Value: IntLiteralNode{Value: 1, Span: SourceSpan{StartLine: 3, StartCol: 4, EndLine: 3, EndCol: 5}},
		Span:  okSpan,
	}
	errSpan := SourceSpan{StartLine: 4, StartCol: 1, EndLine: 4, EndCol: 10}
	errn := ErrExprNode{
		Value: StringLiteralNode{Value: "e", Span: SourceSpan{StartLine: 4, StartCol: 5, EndLine: 4, EndCol: 8}},
		Span:  errSpan,
	}

	t.Run("functionCall", func(t *testing.T) {
		if s := ExpressionSpanStart(call); s != fnSpan {
			t.Fatalf("got %+v want function span %+v", s, fnSpan)
		}
	})
	t.Run("methodCall", func(t *testing.T) {
		if s := ExpressionSpanStart(methodCall); s != recvSpan {
			t.Fatalf("got %+v want receiver span %+v", s, recvSpan)
		}
	})
	t.Run("okExpr", func(t *testing.T) {
		if s := ExpressionSpanStart(ok); s != okSpan {
			t.Fatalf("got %+v want %+v", s, okSpan)
		}
	})
	t.Run("errExpr", func(t *testing.T) {
		if s := ExpressionSpanStart(errn); s != errSpan {
			t.Fatalf("got %+v want %+v", s, errSpan)
		}
	})
}

func TestExpressionSpanStart_nestedIndexAndSliceSuffixes(t *testing.T) {
	t.Parallel()
	fnSpan := SourceSpan{StartLine: 1, StartCol: 2, EndLine: 1, EndCol: 3}
	call := FunctionCallNode{
		Function: Ident{ID: "f", Span: fnSpan},
		CallSpan: SourceSpan{StartLine: 1, StartCol: 3, EndLine: 1, EndCol: 5},
	}
	idx := IndexExpressionNode{
		Target: call,
		Index:  IntLiteralNode{Value: 0, Span: SourceSpan{StartLine: 1, StartCol: 6, EndLine: 1, EndCol: 7}},
	}
	sl := SliceExpressionNode{
		Target: call,
		High:   IntLiteralNode{Value: 1, Span: SourceSpan{StartLine: 1, StartCol: 7, EndLine: 1, EndCol: 8}},
	}
	nestedIdx := IndexExpressionNode{
		Target: idx,
		Index:  IntLiteralNode{Value: 1, Span: FakeSpan()},
	}

	cases := []struct {
		name string
		expr ExpressionNode
	}{
		{name: "indexOfCall", expr: idx},
		{name: "sliceOfCall", expr: sl},
		{name: "nestedIndex", expr: nestedIdx},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExpressionSpanStart(tc.expr)
			if got != fnSpan {
				t.Fatalf("got %+v want callee start %+v", got, fnSpan)
			}
		})
	}
}
