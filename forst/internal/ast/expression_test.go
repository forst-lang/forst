package ast

import (
	"strings"
	"testing"
)

func TestBinaryExpressionNode_String(t *testing.T) {
	b := BinaryExpressionNode{
		Left:     IntLiteralNode{Value: 1},
		Operator: TokenPlus,
		Right:    IntLiteralNode{Value: 2},
	}
	if b.Kind() != NodeKindBinaryExpression || !strings.Contains(b.String(), "PLUS") {
		t.Fatal(b.String())
	}
}

func TestUnaryExpressionNode_String(t *testing.T) {
	u := UnaryExpressionNode{Operator: TokenMinus, Operand: IntLiteralNode{Value: 3}}
	if u.Kind() != NodeKindUnaryExpression || !strings.Contains(u.String(), "MINUS") {
		t.Fatal(u.String())
	}
}

func TestFunctionCallNode_String_multi_arg(t *testing.T) {
	f := FunctionCallNode{
		Function:  Ident{ID: "f"},
		Arguments: []ExpressionNode{IntLiteralNode{Value: 1}, IntLiteralNode{Value: 2}},
	}
	if f.Kind() != NodeKindFunctionCall || !strings.Contains(f.String(), "1") {
		t.Fatal(f.String())
	}
}
