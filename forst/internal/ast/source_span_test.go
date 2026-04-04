package ast

import "testing"

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
