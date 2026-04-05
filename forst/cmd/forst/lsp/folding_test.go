package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestFoldingRangeFromTokenPair_clampsNegativeLineAndColumn(t *testing.T) {
	t.Parallel()
	tokens := []ast.Token{
		{Line: 0, Column: 0, Type: ast.TokenLBrace, Value: "{"},
		{Line: 0, Column: 0, Type: ast.TokenRBrace, Value: "}"},
	}
	m := foldingRangeFromTokenPair(tokens, 0, 1)
	if m["startLine"] != float64(0) && m["startLine"] != 0 {
		t.Fatalf("startLine = %v", m["startLine"])
	}
	if m["startCharacter"] != float64(0) && m["startCharacter"] != 0 {
		t.Fatalf("startCharacter = %v", m["startCharacter"])
	}
}

func TestFoldingRangeFromTokenPair_endCharacterCountsRunes(t *testing.T) {
	t.Parallel()
	tokens := []ast.Token{
		{Line: 1, Column: 1, Type: ast.TokenLBrace, Value: "{"},
		{Line: 2, Column: 1, Type: ast.TokenRBrace, Value: "」"},
	}
	m := foldingRangeFromTokenPair(tokens, 0, 1)
	// One rune wide closing token; LSP end is exclusive past the last rune.
	endChar := m["endCharacter"]
	var ec float64
	switch v := endChar.(type) {
	case float64:
		ec = v
	case int:
		ec = float64(v)
	default:
		t.Fatalf("endCharacter type %T", endChar)
	}
	if ec != 1 {
		t.Fatalf("endCharacter = %v want 1", ec)
	}
}

func TestTokenIndexInSlice(t *testing.T) {
	t.Parallel()
	tokens := []ast.Token{
		{Line: 1, Column: 1, Type: ast.TokenIdentifier, Value: "a"},
		{Line: 1, Column: 3, Type: ast.TokenIdentifier, Value: "b"},
	}
	if tokenIndexInSlice(tokens, nil) != -1 {
		t.Fatal("nil token")
	}
	if tokenIndexInSlice(tokens, &tokens[0]) != 0 {
		t.Fatal("pointer to slice element")
	}
	alias := ast.Token{Line: 1, Column: 1, Type: ast.TokenIdentifier, Value: "a"}
	if tokenIndexInSlice(tokens, &alias) != 0 {
		t.Fatal("value match fallback")
	}
	if tokenIndexInSlice(tokens, &ast.Token{Line: 9, Column: 9}) != -1 {
		t.Fatal("missing")
	}
}

func TestFoldingBracesForFunctionAfterNameToken(t *testing.T) {
	t.Parallel()
	tokens := []ast.Token{
		{Line: 1, Column: 1, Type: ast.TokenFunc, Value: "func"},
		{Line: 1, Column: 6, Type: ast.TokenIdentifier, Value: "main"},
		{Line: 1, Column: 11, Type: ast.TokenLParen, Value: "("},
		{Line: 1, Column: 12, Type: ast.TokenRParen, Value: ")"},
		{Line: 1, Column: 14, Type: ast.TokenLBrace, Value: "{"},
		{Line: 2, Column: 1, Type: ast.TokenRBrace, Value: "}"},
	}
	lb, rb := foldingBracesForFunctionAfterNameToken(tokens, 1)
	if lb != 4 || rb != 5 {
		t.Fatalf("got lb=%d rb=%d", lb, rb)
	}
	lb, rb = foldingBracesForFunctionAfterNameToken(tokens, -1)
	if lb != -1 || rb != -1 {
		t.Fatalf("invalid idx -1: lb=%d rb=%d", lb, rb)
	}
	lb, rb = foldingBracesForFunctionAfterNameToken(tokens, 100)
	if lb != -1 || rb != -1 {
		t.Fatalf("invalid idx past end: lb=%d rb=%d", lb, rb)
	}
}

func TestFoldingBracesForTypeDefAfterNameToken(t *testing.T) {
	t.Parallel()
	tokens := []ast.Token{
		{Line: 1, Column: 1, Type: ast.TokenType, Value: "type"},
		{Line: 1, Column: 6, Type: ast.TokenIdentifier, Value: "Row"},
		{Line: 1, Column: 10, Type: ast.TokenEquals, Value: "="},
		{Line: 1, Column: 12, Type: ast.TokenLBrace, Value: "{"},
		{Line: 2, Column: 3, Type: ast.TokenRBrace, Value: "}"},
	}
	lb, rb := foldingBracesForTypeDefAfterNameToken(tokens, 1)
	if lb != 3 || rb != 4 {
		t.Fatalf("got lb=%d rb=%d", lb, rb)
	}
}

func TestFoldingBracesAfterTypeGuardName_withParens(t *testing.T) {
	t.Parallel()
	tokens := []ast.Token{
		{Line: 1, Column: 1, Type: ast.TokenIs, Value: "is"},
		{Line: 1, Column: 4, Type: ast.TokenLParen, Value: "("},
		{Line: 1, Column: 5, Type: ast.TokenIdentifier, Value: "X"},
		{Line: 1, Column: 6, Type: ast.TokenRParen, Value: ")"},
		{Line: 1, Column: 8, Type: ast.TokenIdentifier, Value: "G"},
		{Line: 1, Column: 10, Type: ast.TokenLBrace, Value: "{"},
		{Line: 2, Column: 1, Type: ast.TokenRBrace, Value: "}"},
	}
	lb, rb := foldingBracesAfterTypeGuardName(tokens, 4)
	if lb != 5 || rb != 6 {
		t.Fatalf("got lb=%d rb=%d", lb, rb)
	}
}

func TestHandleFoldingRange_InvalidParams_ReturnsInvalidParams(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	resp := s.handleFoldingRange(LSPRequest{
		JSONRPC: "2.0",
		ID:      5,
		Params:  json.RawMessage(`{`),
	})
	if resp.Error == nil || resp.Error.Code != -32602 {
		t.Fatalf("got %+v", resp.Error)
	}
}

func TestHandleFoldingRange_parseOk_returnsFunctionBodyRegion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module foldt\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ft := filepath.Join(dir, "fold.ft")
	const src = "package main\n\nfunc main() {\n  var x: Int = 1\n}\n"
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := "file://" + ft
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	params, err := json.Marshal(map[string]interface{}{
		"textDocument": map[string]string{"uri": uri},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp := s.handleFoldingRange(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  json.RawMessage(params),
	})
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	arr, ok := resp.Result.([]interface{})
	if !ok || len(arr) == 0 {
		t.Fatalf("expected folding ranges, got %T %#v", resp.Result, resp.Result)
	}
	m, ok := arr[0].(map[string]interface{})
	if !ok {
		t.Fatalf("range type %T", arr[0])
	}
	sl := m["startLine"]
	if sl != float64(2) && sl != 2 {
		t.Fatalf("startLine = %v (%T)", sl, sl)
	}
	if m["kind"] != "region" {
		t.Fatalf("kind = %v", m["kind"])
	}
}

func TestFoldingRangesForURI_parseError_returnsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "bad.ft")
	uri := "file://" + ft
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = "package main\n\nunexpected\n"
	s.documentMu.Unlock()

	if err := os.WriteFile(ft, []byte("package main\n\nunexpected\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := s.foldingRangesForURI(uri)
	if len(got) != 0 {
		t.Fatalf("expected no ranges on parse error, got %d", len(got))
	}
}

func TestFoldingRangesForURI_typeAndFuncProduceMultipleRanges(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module foldtg\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ft := filepath.Join(dir, "mix.ft")
	const src = `package main

type Row = {
  x: Int
}

func bump(): Int {
  return 1
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := "file://" + ft
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil {
		t.Fatalf("fixture must parse: ok=%v parseErr=%v", ok, ctx.ParseErr)
	}

	got := s.foldingRangesForURI(uri)
	if len(got) < 2 {
		t.Fatalf("want at least type + function folds, got %d: %#v", len(got), got)
	}
}
