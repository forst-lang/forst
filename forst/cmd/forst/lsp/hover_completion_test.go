package lsp

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestTokenSliceIndex_pointerOrValueMatch(t *testing.T) {
	t.Parallel()
	tokens := []ast.Token{
		{Line: 1, Column: 1, Type: ast.TokenIdentifier, Value: "a"},
		{Line: 1, Column: 3, Type: ast.TokenDot, Value: "."},
	}
	if i := tokenSliceIndex(tokens, &tokens[0]); i != 0 {
		t.Fatalf("pointer match: got %d", i)
	}
	alias := ast.Token{Line: 1, Column: 1, Type: ast.TokenIdentifier, Value: "a"}
	if i := tokenSliceIndex(tokens, &alias); i != 0 {
		t.Fatalf("value match: got %d", i)
	}
	if i := tokenSliceIndex(tokens, &ast.Token{Line: 9, Column: 9}); i != -1 {
		t.Fatalf("missing: got %d", i)
	}
}

func TestHoverTextForToken_keyword(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(logrus.New(), false)
	tok := &ast.Token{Type: ast.TokenFunc, Value: "func"}
	if s := hoverTextForToken(tc, nil, tok); s != "`func`" {
		t.Fatalf("got %q", s)
	}
}

func TestHoverTextForToken_stringLiteralNonImportReturnsEmpty(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(logrus.New(), false)
	tok := &ast.Token{Type: ast.TokenStringLiteral, Value: `"hello"`}
	if s := hoverTextForToken(tc, []ast.Token{{Type: ast.TokenStringLiteral, Value: `"hello"`}}, tok); s != "" {
		t.Fatalf("expected no hover for non-import string, got %q", s)
	}
}

func TestHoverTextForToken_intLiteralReturnsEmpty(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(logrus.New(), false)
	tok := &ast.Token{Type: ast.TokenIntLiteral, Value: "42"}
	if s := hoverTextForToken(tc, nil, tok); s != "" {
		t.Fatalf("got %q", s)
	}
}
