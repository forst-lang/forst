package ast

import (
	"strings"
	"testing"
)

func TestReturnNode_String_zero_one_and_multi(t *testing.T) {
	r0 := ReturnNode{}
	if r0.String() != "Return()" {
		t.Fatal(r0.String())
	}
	r1 := ReturnNode{Values: []ExpressionNode{IntLiteralNode{Value: 1}}}
	if !strings.Contains(r1.String(), "1") {
		t.Fatal(r1.String())
	}
	r3 := ReturnNode{Values: []ExpressionNode{
		IntLiteralNode{Value: 1},
		IntLiteralNode{Value: 2},
		IntLiteralNode{Value: 3},
	}}
	s := r3.String()
	if !strings.Contains(s, "1") || !strings.Contains(s, "2") || !strings.Contains(s, "3") {
		t.Fatal(s)
	}
}
