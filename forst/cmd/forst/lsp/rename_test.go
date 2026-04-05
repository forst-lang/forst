package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHandlePrepareRename_ParameterAndBody(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "rn.ft")
	const src = `package main

func id(x Int): Int {
  return x
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil {
		t.Fatalf("analyze: ok=%v parseErr=%v", ok, ctx.ParseErr)
	}
	posUse, ok := lspPosForNthIdentToken(ctx.Tokens, "x", 1)
	if !ok {
		t.Fatal("second x not found")
	}

	resp := s.handlePrepareRename(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/prepareRename",
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]string{"uri": uri},
			"position":     map[string]interface{}{"line": posUse.Line, "character": posUse.Character},
		}),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	res, ok := resp.Result.(LSPPrepareRenameResult)
	if !ok || res.Placeholder != "x" {
		t.Fatalf("result = %#v ok=%v", resp.Result, ok)
	}
}

func TestHandleRename_ParameterAndBody(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "rn.ft")
	const src = `package main

func id(x Int): Int {
  return x
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil {
		t.Fatalf("analyze: ok=%v parseErr=%v", ok, ctx.ParseErr)
	}
	posUse, ok := lspPosForNthIdentToken(ctx.Tokens, "x", 1)
	if !ok {
		t.Fatal("second x not found")
	}

	resp := s.handleRename(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/rename",
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]string{"uri": uri},
			"position":     map[string]interface{}{"line": posUse.Line, "character": posUse.Character},
			"newName":      "y",
		}),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	ws, ok := resp.Result.(LSPWorkspaceEdit)
	if !ok {
		t.Fatalf("result type %T", resp.Result)
	}
	edits := ws.Changes[uri]
	if len(edits) != 2 {
		t.Fatalf("want 2 edits, got %d", len(edits))
	}
	for _, e := range edits {
		if e.NewText != "y" {
			t.Fatalf("newText = %q", e.NewText)
		}
	}
}

func TestHandleRename_InvalidNewName(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "rn.ft")
	const src = `package main

func id(x Int): Int {
  return x
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil {
		t.Fatalf("analyze: ok=%v parseErr=%v", ok, ctx.ParseErr)
	}
	posUse, ok := lspPosForNthIdentToken(ctx.Tokens, "x", 1)
	if !ok {
		t.Fatal("second x not found")
	}

	resp := s.handleRename(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]string{"uri": uri},
			"position":     map[string]interface{}{"line": posUse.Line, "character": posUse.Character},
			"newName":      "bad name",
		}),
	})
	if resp.Error == nil || resp.Error.Code != -32602 {
		t.Fatalf("expected invalid params, got %+v", resp.Error)
	}
}

func TestIsValidForstIdentifierRename(t *testing.T) {
	t.Parallel()
	if !isValidForstIdentifierRename("x") || !isValidForstIdentifierRename("_foo") || !isValidForstIdentifierRename("Foo2") {
		t.Fatal("expected valid")
	}
	if isValidForstIdentifierRename("") || isValidForstIdentifierRename("2a") || isValidForstIdentifierRename("a-b") {
		t.Fatal("expected invalid")
	}
}

func TestHandlePrepareRename_ResultJSONShape(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "rn.ft")
	const src = `package main

func id(x Int): Int {
  return x
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil {
		t.Fatalf("analyze: ok=%v parseErr=%v", ok, ctx.ParseErr)
	}
	posUse, ok := lspPosForNthIdentToken(ctx.Tokens, "x", 1)
	if !ok {
		t.Fatal("second x not found")
	}

	resp := s.handlePrepareRename(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]string{"uri": uri},
			"position":     map[string]interface{}{"line": posUse.Line, "character": posUse.Character},
		}),
	})
	b, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["placeholder"] != "x" {
		t.Fatalf("placeholder = %v", m["placeholder"])
	}
}
