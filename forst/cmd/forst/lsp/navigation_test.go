package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

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
	for _, s := range syms {
		names[s.Name] = s.Kind
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
