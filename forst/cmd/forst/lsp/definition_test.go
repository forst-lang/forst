package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestFindFuncNameToken_topLevelOnly(t *testing.T) {
	t.Parallel()
	tokens := []ast.Token{
		{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
		{Type: ast.TokenIdentifier, Value: "outer", Line: 1, Column: 6},
		{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 20},
		{Type: ast.TokenFunc, Value: "func", Line: 2, Column: 3},
		{Type: ast.TokenIdentifier, Value: "inner", Line: 2, Column: 8},
	}
	if got := findFuncNameToken(tokens, "outer"); got == nil || got.Value != "outer" {
		t.Fatalf("outer: got %+v", got)
	}
	if got := findFuncNameToken(tokens, "inner"); got != nil {
		t.Fatalf("inner should not match inside braces, got %+v", got)
	}
}

func TestFindTypeGuardNameToken_ignoresEnsureInsideFunction(t *testing.T) {
	t.Parallel()
	tokens := []ast.Token{
		{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
		{Type: ast.TokenIdentifier, Value: "f", Line: 1, Column: 6},
		{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 7},
		{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 8},
		{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 10},
		{Type: ast.TokenEnsure, Value: "ensure", Line: 2, Column: 3},
		{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 10},
		{Type: ast.TokenIs, Value: "is", Line: 2, Column: 12},
		{Type: ast.TokenIdentifier, Value: "T", Line: 2, Column: 15},
		{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 1},
		{Type: ast.TokenIs, Value: "is", Line: 5, Column: 1},
		{Type: ast.TokenLParen, Value: "(", Line: 5, Column: 4},
		{Type: ast.TokenIdentifier, Value: "p", Line: 5, Column: 5},
		{Type: ast.TokenColon, Value: ":", Line: 5, Column: 6},
		{Type: ast.TokenString, Value: "String", Line: 5, Column: 8},
		{Type: ast.TokenRParen, Value: ")", Line: 5, Column: 14},
		{Type: ast.TokenIdentifier, Value: "G", Line: 5, Column: 16},
	}
	if got := findTypeGuardNameToken(tokens, "G"); got == nil || got.Value != "G" {
		t.Fatalf("expected G at col 16, got %+v", got)
	}
}

func TestHandleDefinition_Function(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)

	dir := t.TempDir()
	ftPath := filepath.Join(dir, "def.ft")
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

	resp := s.handleDefinition(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/definition",
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"position":     map[string]interface{}{"line": 7, "character": 2},
		}),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	loc := mustLSPLocation(t, resp.Result)
	if loc.Range.Start.Line != 2 || loc.Range.Start.Character != 5 {
		t.Fatalf("expected func bump at line 3 col 6, got %+v", loc.Range.Start)
	}
}

func TestHandleDefinition_TypeAlias(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())

	dir := t.TempDir()
	ftPath := filepath.Join(dir, "def.ft")
	const src = `package main

type Row = {
  x: Int
}

func useRow(r Row): Int {
  return 0
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := "file://" + ftPath

	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	resp := s.handleDefinition(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/definition",
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"position":     map[string]interface{}{"line": 6, "character": 14},
		}),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	loc := mustLSPLocation(t, resp.Result)
	if loc.Range.Start.Line != 2 || loc.Range.Start.Character != 5 {
		t.Fatalf("expected type Row at line 3 col 6, got %+v", loc.Range.Start)
	}
}

func mustJSONParams(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustLSPLocation(t *testing.T, v interface{}) LSPLocation {
	t.Helper()
	switch x := v.(type) {
	case LSPLocation:
		return x
	case *LSPLocation:
		if x == nil {
			t.Fatal("nil *LSPLocation")
		}
		return *x
	default:
		t.Fatalf("expected LSPLocation, got %T %#v", v, v)
		panic("unreachable")
	}
}
