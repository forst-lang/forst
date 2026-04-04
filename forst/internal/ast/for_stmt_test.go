package ast

import "testing"

func TestBreakNode_String_withLabel(t *testing.T) {
	t.Parallel()
	b := BreakNode{Label: &Ident{ID: "outer"}}
	if b.String() != "Break(outer)" {
		t.Fatalf("got %q", b.String())
	}
}

func TestContinueNode_String_withLabel(t *testing.T) {
	t.Parallel()
	c := ContinueNode{Label: &Ident{ID: "inner"}}
	if c.String() != "Continue(inner)" {
		t.Fatalf("got %q", c.String())
	}
}
