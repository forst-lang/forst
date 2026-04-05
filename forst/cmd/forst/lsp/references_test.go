package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
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

func TestHandleReferences_excludesDeclaration(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "ref.ft")
	const src = `package main

func bump(): Int {
  return 1
}

func main() {
  bump()
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := "file://" + ftPath
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     map[string]interface{}{"line": 7, "character": 2},
		"context":      map[string]interface{}{"includeDeclaration": false},
	}
	resp := s.handleReferences(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/references",
		Params:  mustJSONParams(t, params),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	locs, ok := resp.Result.([]LSPLocation)
	if !ok {
		t.Fatalf("result type %T", resp.Result)
	}
	if len(locs) != 1 {
		t.Fatalf("want 1 ref (call site only), got %d", len(locs))
	}
	if locs[0].Range.Start.Line != 7 {
		t.Fatalf("expected call on line 8 (0-based 7), got %+v", locs[0].Range.Start)
	}
}
