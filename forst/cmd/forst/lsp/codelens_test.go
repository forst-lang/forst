package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHandleCodeLens_referenceCountsAcrossMergedPackage(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)

	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.ft")
	bPath := filepath.Join(dir, "b.ft")
	const srcA = `package main

func foo(): Int {
  return 1
}
`
	const srcB = `package main

func bar(): Int {
  return foo()
}
`
	if err := os.WriteFile(aPath, []byte(srcA), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bPath, []byte(srcB), 0o644); err != nil {
		t.Fatal(err)
	}
	uriA := mustFileURI(t, aPath)
	uriB := mustFileURI(t, bPath)

	s.documentMu.Lock()
	s.openDocuments[uriA] = srcA
	s.openDocuments[uriB] = srcB
	s.documentMu.Unlock()

	resp := s.handleCodeLens(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/codeLens",
		Params:  mustJSONParams(t, map[string]interface{}{"textDocument": map[string]string{"uri": uriA}}),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	raw, ok := resp.Result.([]interface{})
	if !ok {
		t.Fatalf("result type %T", resp.Result)
	}
	var fooLens map[string]interface{}
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		cmd, _ := m["command"].(map[string]interface{})
		if cmd == nil {
			continue
		}
		if cmd["title"] == "2 references" {
			fooLens = m
			break
		}
	}
	if fooLens == nil {
		t.Fatalf("expected a 2 references lens on foo, got %#v", raw)
	}
}

func TestHandleCodeLens_emptyOnParseError(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	uri := mustFileURI(t, filepath.Join(t.TempDir(), "bad.ft"))
	s.documentMu.Lock()
	s.openDocuments[uri] = `package main
func broken {`
	s.documentMu.Unlock()

	resp := s.handleCodeLens(LSPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "textDocument/codeLens",
		Params:  mustJSONParams(t, map[string]interface{}{"textDocument": map[string]string{"uri": uri}}),
	})
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	raw, ok := resp.Result.([]interface{})
	if !ok {
		t.Fatalf("result type %T", resp.Result)
	}
	if len(raw) != 0 {
		t.Fatalf("expected no lenses on parse error, got %d", len(raw))
	}
}
