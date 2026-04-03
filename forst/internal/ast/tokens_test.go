package ast

import "testing"

func TestTokenIdent_predicates(t *testing.T) {
	if !TokenPlus.IsBinaryOperator() || !TokenPlus.IsArithmeticBinaryOperator() {
		t.Fatal("TokenPlus")
	}
	if !TokenEquals.IsComparisonBinaryOperator() || !TokenEquals.IsBinaryOperator() {
		t.Fatal("TokenEquals")
	}
	if !TokenLogicalAnd.IsLogicalBinaryOperator() || !TokenLogicalAnd.IsBinaryOperator() {
		t.Fatal("TokenLogicalAnd")
	}
	if !TokenIs.IsBinaryOperator() {
		t.Fatal("TokenIs should count as binary op for parser")
	}
	if !TokenLogicalNot.IsUnaryOperator() {
		t.Fatal()
	}
	if !TokenIntLiteral.IsLiteral() || !TokenTrue.IsLiteral() {
		t.Fatal()
	}
	if TokenImport.IsLiteral() || TokenImport.IsBinaryOperator() {
		t.Fatal("TokenImport should not be literal/binary")
	}
}

func TestTokenIdent_String_default_branch(t *testing.T) {
	if TokenImport.String() != string(TokenImport) {
		t.Fatalf("%q", TokenImport.String())
	}
}

func TestTokenIdent_String_logical_and_bitwise(t *testing.T) {
	if TokenBitwiseAnd.String() != "&" || TokenBitwiseOr.String() != "|" {
		t.Fatal(TokenBitwiseAnd.String(), TokenBitwiseOr.String())
	}
	if TokenLogicalAnd.String() != "&&" || TokenLogicalOr.String() != "||" {
		t.Fatal(TokenLogicalAnd.String(), TokenLogicalOr.String())
	}
	if TokenPlus.String() != string(TokenPlus) {
		t.Fatalf("%q", TokenPlus.String())
	}
}
