package lsp

import (
	"testing"

	"forst/internal/ast"
)

func TestSymbolsFromParsedDocument_empty(t *testing.T) {
	t.Parallel()
	syms := symbolsFromParsedDocument("file:///a.ft", nil, nil)
	if len(syms) != 0 {
		t.Fatalf("expected no symbols, got %d", len(syms))
	}
}

func TestSymbolsFromParsedDocument_skipsHashTypeNames(t *testing.T) {
	t.Parallel()
	nodes := []ast.Node{
		ast.TypeDefNode{Ident: ast.TypeIdent("T_abc123")},
	}
	toks := []ast.Token{
		{Line: 1, Column: 1, Type: ast.TokenType},
		{Line: 1, Column: 6, Type: ast.TokenIdentifier, Value: "T_abc123"},
	}
	syms := symbolsFromParsedDocument("file:///b.ft", toks, nodes)
	if len(syms) != 0 {
		t.Fatalf("expected hash-backed type name skipped, got %#v", syms)
	}
}
