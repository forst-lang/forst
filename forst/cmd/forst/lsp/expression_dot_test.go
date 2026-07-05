package lsp

import (
	"testing"

	"forst/internal/ast"
)

func TestReceiverExpressionSourceBeforeDot_qualifiedCall(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "time", Line: 1, Column: 1},
		{Type: ast.TokenDot, Value: ".", Line: 1, Column: 5},
		{Type: ast.TokenIdentifier, Value: "Now", Line: 1, Column: 6},
		{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
		{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
		{Type: ast.TokenDot, Value: ".", Line: 1, Column: 11},
		{Type: ast.TokenIdentifier, Value: "Format", Line: 1, Column: 12},
	}
	src, ok := receiverExpressionSourceBeforeDot(tokens, 5)
	if !ok {
		t.Fatal("expected ok")
	}
	if src != "time.Now()" {
		t.Fatalf("got %q", src)
	}
}

func TestReceiverExpressionSourceBeforeDot_simpleIdent(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 1},
		{Type: ast.TokenDot, Value: ".", Line: 1, Column: 2},
		{Type: ast.TokenIdentifier, Value: "y", Line: 1, Column: 3},
	}
	src, ok := receiverExpressionSourceBeforeDot(tokens, 1)
	if !ok || src != "x" {
		t.Fatalf("got %q ok=%v", src, ok)
	}
}
