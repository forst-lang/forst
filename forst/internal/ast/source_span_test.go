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
