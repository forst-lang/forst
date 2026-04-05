package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
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

func TestSymbolsFromParsedDocument_funcAndType(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "sym.ft")
	const src = `package main

type Row = {
  x: Int
}

func bump(): Int {
  return 1
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := "file://" + ftPath
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil {
		t.Fatalf("analyze: ok=%v parseErr=%v", ok, ctx.ParseErr)
	}
	syms := symbolsFromParsedDocument(uri, ctx.Tokens, ctx.Nodes)
	if len(syms) != 2 {
		t.Fatalf("want 2 symbols, got %d %#v", len(syms), syms)
	}
	names := map[string]int{}
	for _, sym := range syms {
		names[sym.Name] = sym.Kind
	}
	if names["Row"] != lspSymbolKindStruct {
		t.Fatalf("Row kind: %v", names["Row"])
	}
	if names["bump"] != lspSymbolKindFunction {
		t.Fatalf("bump kind: %v", names["bump"])
	}
}

func TestHandleDocumentSymbol_JSON(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "doc.ft")
	const src = `package main

func alpha(): Int { return 0 }
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := "file://" + ftPath
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	resp := s.handleDocumentSymbol(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/documentSymbol",
		Params:  mustJSONParams(t, map[string]interface{}{"textDocument": map[string]interface{}{"uri": uri}}),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	raw, ok := resp.Result.([]LspSymbolInformation)
	if !ok {
		t.Fatalf("result type %T", resp.Result)
	}
	if len(raw) != 1 || raw[0].Name != "alpha" {
		t.Fatalf("got %#v", raw)
	}
}
