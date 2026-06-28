package ast

import "testing"

func TestIsCompoundAssignToken(t *testing.T) {
	t.Parallel()
	if !IsCompoundAssignToken(TokenPlusEq) || !IsCompoundAssignToken(TokenMinusEq) {
		t.Fatal("expected += and -= to be compound assign tokens")
	}
	if IsCompoundAssignToken(TokenPlus) || IsCompoundAssignToken(TokenEquals) {
		t.Fatal("plain operators must not be compound assign tokens")
	}
}

func TestCompoundAssignBinaryOp(t *testing.T) {
	t.Parallel()
	op, ok := CompoundAssignBinaryOp(TokenPlusEq)
	if !ok || op != TokenPlus {
		t.Fatalf("TokenPlusEq -> %+v ok=%v", op, ok)
	}
	if _, ok := CompoundAssignBinaryOp(TokenPlus); ok {
		t.Fatal("TokenPlus should not map to compound binary op")
	}
}

func TestIsAssignmentOperatorToken(t *testing.T) {
	t.Parallel()
	if !IsAssignmentOperatorToken(Token{Type: TokenEquals, Value: "="}) {
		t.Fatal("expected plain = to be assignment operator")
	}
	if IsAssignmentOperatorToken(Token{Type: TokenEquals, Value: "=="}) {
		t.Fatal("== must not be assignment operator")
	}
	if !IsAssignmentOperatorToken(Token{Type: TokenPlusEq, Value: "+="}) {
		t.Fatal("expected += to be assignment operator")
	}
}

func TestAssignmentNode_String_compound(t *testing.T) {
	t.Parallel()
	a := AssignmentNode{
		LValues:    []ExpressionNode{VariableNode{Ident: Ident{ID: "n"}}},
		RValues:    []ExpressionNode{IntLiteralNode{Value: 5}},
		CompoundOp: TokenPlusEq,
	}
	if s := a.String(); s != "Assignment(Variable(n) += 5)" {
		t.Fatalf("got %q", s)
	}
}
