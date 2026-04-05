package printer

import (
	"testing"

	"forst/internal/ast"
)

func TestTokenBinary_mapsOperatorTokens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		op   ast.TokenIdent
		want string
	}{
		{"plus", ast.TokenPlus, "+"},
		{"minus", ast.TokenMinus, "-"},
		{"star", ast.TokenStar, "*"},
		{"divide", ast.TokenDivide, "/"},
		{"modulo", ast.TokenModulo, "%"},
		{"equals", ast.TokenEquals, "=="},
		{"notEquals", ast.TokenNotEquals, "!="},
		{"greater", ast.TokenGreater, ">"},
		{"less", ast.TokenLess, "<"},
		{"greaterEqual", ast.TokenGreaterEqual, ">="},
		{"lessEqual", ast.TokenLessEqual, "<="},
		{"logicalAnd", ast.TokenLogicalAnd, "&&"},
		{"logicalOr", ast.TokenLogicalOr, "||"},
		{"is", ast.TokenIs, "is"},
		{"bitwiseAnd", ast.TokenBitwiseAnd, "&"},
		{"bitwiseOr", ast.TokenBitwiseOr, "|"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tokenBinary(tt.op); got != tt.want {
				t.Fatalf("tokenBinary(%q) = %q, want %q", tt.op, got, tt.want)
			}
		})
	}
}

func TestTokenBinary_unknownFallsBackToTokenString(t *testing.T) {
	t.Parallel()
	op := ast.TokenIdent("customOp")
	if got := tokenBinary(op); got != "customOp" {
		t.Fatalf("got %q", got)
	}
}

func TestTokenUnary_mapsOperatorTokens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		op   ast.TokenIdent
		want string
	}{
		{"not", ast.TokenLogicalNot, "!"},
		{"minus", ast.TokenMinus, "-"},
		{"star", ast.TokenStar, "*"},
		{"bitwiseAnd", ast.TokenBitwiseAnd, "&"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tokenUnary(tt.op); got != tt.want {
				t.Fatalf("tokenUnary(%q) = %q, want %q", tt.op, got, tt.want)
			}
		})
	}
}

func TestTokenUnary_unknownFallsBackToTokenString(t *testing.T) {
	t.Parallel()
	op := ast.TokenIdent("x")
	if got := tokenUnary(op); got != "x" {
		t.Fatalf("got %q", got)
	}
}
