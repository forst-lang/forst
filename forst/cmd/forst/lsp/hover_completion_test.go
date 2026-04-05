package lsp

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLexicalHoverMarkdown_keywordAndIdentifier(t *testing.T) {
	t.Parallel()
	if s := lexicalHoverMarkdown(&ast.Token{Type: ast.TokenFunc, Value: "func"}); s != "`func`" {
		t.Fatalf("keyword: got %q", s)
	}
	id := &ast.Token{Type: ast.TokenIdentifier, Value: "foo"}
	if s := lexicalHoverMarkdown(id); !strings.Contains(s, "`foo`") || !strings.Contains(s, "parses") {
		t.Fatalf("identifier: got %q", s)
	}
	if s := lexicalHoverMarkdown(&ast.Token{Type: ast.TokenIntLiteral, Value: "1"}); s != "" {
		t.Fatalf("literal: got %q", s)
	}
}

func TestFindHoverForPosition_parseError_keywordHover(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "bad_hover.ft")
	// Same top-level rejection as analyze_test (parser error, tokens still present).
	const src = "package main\n\nunexpected\n"
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := "file://" + ft
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr == nil {
		t.Fatal("expected parse error for test fixture")
	}

	// Line 0: `package` keyword
	hPkg := s.findHoverForPosition(uri, LSPPosition{Line: 0, Character: 2})
	if hPkg == nil || hPkg.Contents.Value != "`package`" {
		if hPkg == nil {
			t.Fatal("expected keyword hover on package when parse fails")
		}
		t.Fatalf("package hover: got %q", hPkg.Contents.Value)
	}
	// Line 2: `unexpected` — lexical identifier hover
	h := s.findHoverForPosition(uri, LSPPosition{Line: 2, Character: 2})
	if h == nil || !strings.Contains(h.Contents.Value, "`unexpected`") {
		if h == nil {
			t.Fatal("expected hover when parse fails")
		}
		t.Fatalf("identifier hover: got %q", h.Contents.Value)
	}
	_ = ctx
}

func TestHandleWorkspaceSymbol_filtersQuery(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "ws.ft")
	const src = `package main

func fooBar(): Int { return 0 }
func other(): Int { return 1 }
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := "file://" + ftPath
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	resp := s.handleWorkspaceSymbol(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "workspace/symbol",
		Params:  mustJSONParams(t, map[string]interface{}{"query": "bar"}),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	syms, ok := resp.Result.([]LspSymbolInformation)
	if !ok {
		t.Fatalf("result type %T", resp.Result)
	}
	if len(syms) != 1 || syms[0].Name != "fooBar" {
		t.Fatalf("got %#v", syms)
	}
}
