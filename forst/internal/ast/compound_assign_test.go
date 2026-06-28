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

func TestCompoundAssignBinaryOp_allOperators(t *testing.T) {
	t.Parallel()
	cases := []struct {
		compound TokenIdent
		binary   TokenIdent
	}{
		{TokenMinusEq, TokenMinus},
		{TokenStarEq, TokenStar},
		{TokenDivideEq, TokenDivide},
		{TokenModuloEq, TokenModulo},
		{TokenBitwiseAndEq, TokenBitwiseAnd},
		{TokenBitwiseOrEq, TokenBitwiseOr},
	}
	for _, tc := range cases {
		op, ok := CompoundAssignBinaryOp(tc.compound)
		if !ok || op != tc.binary {
			t.Fatalf("%s -> %v ok=%v", tc.compound, op, ok)
		}
	}
	if _, ok := CompoundAssignBinaryOp(TokenColonEquals); ok {
		t.Fatal(":= is not compound assign")
	}
}

func TestCompoundAssignOperatorString_allSpellings(t *testing.T) {
	t.Parallel()
	cases := map[TokenIdent]string{
		TokenPlusEq:       "+=",
		TokenMinusEq:      "-=",
		TokenStarEq:       "*=",
		TokenDivideEq:     "/=",
		TokenModuloEq:     "%=",
		TokenBitwiseAndEq: "&=",
		TokenBitwiseOrEq:  "|=",
	}
	for tok, want := range cases {
		if got := CompoundAssignOperatorString(tok); got != want {
			t.Fatalf("%s: got %q want %q", tok, got, want)
		}
	}
	if got := CompoundAssignOperatorString(TokenPlus); got != string(TokenPlus) {
		t.Fatalf("unknown token: %q", got)
	}
}

func TestIsAssignmentOperatorToken_colonEquals(t *testing.T) {
	t.Parallel()
	if !IsAssignmentOperatorToken(Token{Type: TokenColonEquals, Value: ":="}) {
		t.Fatal("expected := to be assignment operator")
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
