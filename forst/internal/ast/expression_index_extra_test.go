package ast

import "testing"

func TestIndexExpressionNode_String_and_Kind(t *testing.T) {
	t.Parallel()
	i := IndexExpressionNode{
		Target: IntLiteralNode{Value: 1},
		Index:  IntLiteralNode{Value: 0},
	}
	if i.Kind() != NodeKindIndexExpression {
		t.Fatalf("Kind: %v", i.Kind())
	}
	if i.String() == "" {
		t.Fatal("expected non-empty String")
	}
}
