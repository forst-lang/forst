package ast

import (
	"strings"
	"testing"
)

func TestAssignmentNode_String_short_with_explicit_types(t *testing.T) {
	tInt := NewBuiltinType(TypeInt)
	a := AssignmentNode{
		LValues: []ExpressionNode{
			VariableNode{Ident: Ident{ID: "a"}},
			VariableNode{Ident: Ident{ID: "b"}},
		},
		RValues: []ExpressionNode{
			IntLiteralNode{Value: 1},
			IntLiteralNode{Value: 2},
		},
		ExplicitTypes: []*TypeNode{&tInt, &tInt},
		IsShort:       true,
	}
	if !strings.Contains(a.String(), ":=") || !strings.Contains(a.String(), "a") {
		t.Fatal(a.String())
	}
}
