package ast

import "testing"

func TestIsUnaryAmpersandAt_afterColonEquals(t *testing.T) {
	tokens := []Token{
		{Type: TokenIdentifier, Value: "p"},
		{Type: TokenColonEquals, Value: ":="},
		{Type: TokenBitwiseAnd, Value: "&"},
		{Type: TokenIdentifier, Value: "x"},
	}
	if !IsUnaryAmpersandAt(tokens, 2) {
		t.Fatal(":= &x should be unary address-of")
	}
}

func TestIsUnaryAmpersandAt_afterLParen(t *testing.T) {
	tokens := []Token{
		{Type: TokenLParen, Value: "("},
		{Type: TokenBitwiseAnd, Value: "&"},
		{Type: TokenIdentifier, Value: "x"},
		{Type: TokenRParen, Value: ")"},
	}
	if !IsUnaryAmpersandAt(tokens, 1) {
		t.Fatal("( &x ) should be unary address-of")
	}
}

func TestIsUnaryAmpersandAt_afterReturn(t *testing.T) {
	tokens := []Token{
		{Type: TokenReturn, Value: "return"},
		{Type: TokenBitwiseAnd, Value: "&"},
		{Type: TokenIdentifier, Value: "x"},
	}
	if !IsUnaryAmpersandAt(tokens, 1) {
		t.Fatal("return &x should be unary address-of")
	}
}

func TestIsUnaryAmpersandAt_binaryBetweenIdentifiers(t *testing.T) {
	tokens := []Token{
		{Type: TokenIdentifier, Value: "a"},
		{Type: TokenBitwiseAnd, Value: "&"},
		{Type: TokenIdentifier, Value: "b"},
	}
	if IsUnaryAmpersandAt(tokens, 1) {
		t.Fatal("a & b should be binary bitwise AND")
	}
}

func TestIsUnaryAmpersandAt_binaryAfterRParen(t *testing.T) {
	tokens := []Token{
		{Type: TokenLParen, Value: "("},
		{Type: TokenIdentifier, Value: "a"},
		{Type: TokenRParen, Value: ")"},
		{Type: TokenBitwiseAnd, Value: "&"},
		{Type: TokenIdentifier, Value: "b"},
	}
	if IsUnaryAmpersandAt(tokens, 3) {
		t.Fatal("(a) & b should be binary bitwise AND")
	}
}

func TestIsUnaryAmpersandAt_skipsComment(t *testing.T) {
	tokens := []Token{
		{Type: TokenColonEquals, Value: ":="},
		{Type: TokenComment, Value: "// take addr"},
		{Type: TokenBitwiseAnd, Value: "&"},
		{Type: TokenIdentifier, Value: "x"},
	}
	if !IsUnaryAmpersandAt(tokens, 2) {
		t.Fatal(":= /*comment*/ &x should still be unary")
	}
}

func TestIsUnaryAmpersandAt_startOfFile(t *testing.T) {
	tokens := []Token{
		{Type: TokenBitwiseAnd, Value: "&"},
		{Type: TokenIdentifier, Value: "x"},
	}
	if !IsUnaryAmpersandAt(tokens, 0) {
		t.Fatal("& at start of file should be unary")
	}
}

func TestIsUnaryAmpersandAt_wrongToken(t *testing.T) {
	tokens := []Token{{Type: TokenPlus, Value: "+"}}
	if IsUnaryAmpersandAt(tokens, 0) {
		t.Fatal("non-& token must not report unary ampersand")
	}
	if IsUnaryAmpersandAt(tokens, -1) || IsUnaryAmpersandAt(nil, 0) {
		t.Fatal("out of range must be false")
	}
}
