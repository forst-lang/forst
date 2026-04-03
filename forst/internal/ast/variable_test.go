package ast

import (
	"strings"
	"testing"
)

func TestVariableNode_String_explicit_and_plain(t *testing.T) {
	v := VariableNode{Ident: Ident{ID: "x"}, ExplicitType: NewBuiltinType(TypeInt)}
	if !strings.Contains(v.String(), "TYPE_INT") && !strings.Contains(v.String(), "x") {
		t.Fatal(v.String())
	}
	if (VariableNode{Ident: Ident{ID: "y"}}).String() != "Variable(y)" {
		t.Fatal()
	}
}

func TestVariableNode_isValue_marker(t *testing.T) {
	VariableNode{}.isValue()
}
