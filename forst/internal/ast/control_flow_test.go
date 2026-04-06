package ast

import (
	"strings"
	"testing"
)

func TestIfNode_String_with_else_and_init(t *testing.T) {
	ifn := IfNode{
		Init:      AssignmentNode{LValues: []ExpressionNode{VariableNode{Ident: Ident{ID: "i"}}}, RValues: []ExpressionNode{IntLiteralNode{Value: 0}}},
		Condition: BoolLiteralNode{Value: true},
		Body:      []Node{},
		ElseIfs:   []ElseIfNode{{Condition: BoolLiteralNode{Value: false}, Body: []Node{}}},
		Else:      &ElseBlockNode{Body: []Node{IntLiteralNode{Value: 1}}},
	}
	s := ifn.String()
	if !strings.Contains(s, "If(") || !strings.Contains(s, "Else") {
		t.Fatal(s)
	}
}

func TestElseIfNode_and_ElseBlockNode_kind(t *testing.T) {
	ei := ElseIfNode{Condition: BoolLiteralNode{Value: true}, Body: []Node{}}
	if ei.Kind() != NodeKindElseIf {
		t.Fatal(ei.Kind())
	}
	eb := ElseBlockNode{Body: []Node{}}
	if eb.Kind() != NodeKindElseBlock {
		t.Fatal(eb.Kind())
	}
}
