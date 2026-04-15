package lsp

import (
	"testing"

	"forst/internal/ast"
)

func TestCompletionHelpers_findFirstTokenAndURIBasename(t *testing.T) {
	tokens := lexTokensForLSPHelperTest(`package main

func f() {
  if true {
    return
  }
}
`)

	ifIndex := findFirstToken(tokens, 0, len(tokens), ast.TokenIf)
	if ifIndex < 0 {
		t.Fatal("expected to find if token")
	}
	missingIndex := findFirstToken(tokens, 0, len(tokens), ast.TokenEnsure)
	if missingIndex != -1 {
		t.Fatalf("expected missing ensure token to return -1, got %d", missingIndex)
	}

	base := uriDisplayBasename("file:///tmp/project/file.ft")
	if base != "file.ft" {
		t.Fatalf("unexpected uri basename: %q", base)
	}
}

func TestCompletionHelpers_elseIfElseAndEnsureBraces(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenElseIf},
		{Type: ast.TokenLParen},
		{Type: ast.TokenIdentifier, Value: "x"},
		{Type: ast.TokenRParen},
		{Type: ast.TokenLBrace},
		{Type: ast.TokenIdentifier, Value: "_"},
		{Type: ast.TokenRBrace},
		{Type: ast.TokenElse},
		{Type: ast.TokenLBrace},
		{Type: ast.TokenIdentifier, Value: "_"},
		{Type: ast.TokenRBrace},
		{Type: ast.TokenEnsure},
		{Type: ast.TokenIdentifier, Value: "x"},
		{Type: ast.TokenLBrace},
		{Type: ast.TokenIdentifier, Value: "y"},
		{Type: ast.TokenRBrace},
	}

	elseIfIndex := nthTokenIndexByType(tokens, ast.TokenElseIf, 0)
	elseIfL, elseIfR := elseIfThenBraces(tokens, elseIfIndex)
	if elseIfL < 0 || elseIfR <= elseIfL {
		t.Fatalf("expected valid else-if brace pair, got l=%d r=%d", elseIfL, elseIfR)
	}

	elseIndex := nthTokenIndexByType(tokens, ast.TokenElse, 0)
	if elseIndex < 0 {
		t.Fatal("expected else token")
	}
	elseL, elseR := elseBlockBraces(tokens, elseIndex)
	if elseL < 0 || elseR <= elseL {
		t.Fatalf("expected valid else brace pair, got l=%d r=%d", elseL, elseR)
	}

	ensureIndex := nthTokenIndexByType(tokens, ast.TokenEnsure, 0)
	if ensureIndex < 0 {
		t.Fatal("expected ensure token")
	}
	ensureL, ensureR := ensureBlockBraces(tokens, ensureIndex)
	if ensureL < 0 || ensureR <= ensureL {
		t.Fatalf("expected valid ensure block brace pair, got l=%d r=%d", ensureL, ensureR)
	}
}
