package lexer

import (
	"testing"

	"forst/internal/ast"
)

func TestGetTokenType_keywordsAndOperators(t *testing.T) {
	tests := []struct {
		word string
		want ast.TokenIdent
	}{
		{"func", ast.TokenFunc},
		{"ensure", ast.TokenEnsure},
		{"else if", ast.TokenElseIf},
		{"==", ast.TokenEquals},
		{":=", ast.TokenColonEquals},
		{"->", ast.TokenArrow},
		{"//", ast.TokenComment},
		{".", ast.TokenDot},
		{"++", ast.TokenPlusPlus},
		{"--", ast.TokenMinusMinus},
		{"true", ast.TokenTrue},
		{"false", ast.TokenFalse},
	}
	for _, tt := range tests {
		if got := GetTokenType(tt.word); got != tt.want {
			t.Errorf("GetTokenType(%q) = %v, want %v", tt.word, got, tt.want)
		}
	}
}

func TestGetTokenType_numericAndIdentifier(t *testing.T) {
	if got := GetTokenType("42"); got != ast.TokenIntLiteral {
		t.Fatalf("int literal: got %v", got)
	}
	if got := GetTokenType("3.14"); got != ast.TokenFloatLiteral {
		t.Fatalf("float literal: got %v", got)
	}
	if got := GetTokenType("1e2"); got != ast.TokenFloatLiteral {
		t.Fatalf("exponent float: got %v", got)
	}
	if got := GetTokenType("foo"); got != ast.TokenIdentifier {
		t.Fatalf("identifier: got %v", got)
	}
}
