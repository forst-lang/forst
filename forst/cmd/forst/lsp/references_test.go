package lsp

import (
	"testing"

	"forst/internal/ast"
)

func TestTokenSamePosition(t *testing.T) {
	t.Parallel()
	a := &ast.Token{Line: 3, Column: 5, Type: ast.TokenIdentifier, Value: "x"}
	b := &ast.Token{Line: 3, Column: 5, Type: ast.TokenIdentifier, Value: "y"}
	if !tokenSamePosition(a, b) {
		t.Fatal("same line/col should match regardless of value")
	}
	if tokenSamePosition(a, nil) {
		t.Fatal("nil b")
	}
	if tokenSamePosition(nil, b) {
		t.Fatal("nil a")
	}
	c := &ast.Token{Line: 3, Column: 6}
	if tokenSamePosition(a, c) {
		t.Fatal("different column should not match")
	}
}

func TestCollectIdentifierReferences_includeDecl(t *testing.T) {
	t.Parallel()
	toks := []ast.Token{
		{Line: 1, Column: 1, Type: ast.TokenIdentifier, Value: "Foo"},
		{Line: 2, Column: 1, Type: ast.TokenIdentifier, Value: "Foo"},
	}
	def := &toks[0]
	uri := "file:///x.ft"
	with := collectIdentifierReferences(uri, toks, "Foo", def, true)
	if len(with) != 2 {
		t.Fatalf("includeDecl: want 2 refs, got %d", len(with))
	}
	without := collectIdentifierReferences(uri, toks, "Foo", def, false)
	if len(without) != 1 {
		t.Fatalf("exclude decl: want 1 ref, got %d", len(without))
	}
}
