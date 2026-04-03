package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestIdentifierPrefixAt(t *testing.T) {
	content := "  hello"
	got := identifierPrefixAt(content, LSPPosition{Line: 0, Character: 7})
	if got != "hello" {
		t.Fatalf("got %q", got)
	}
}

func TestLhsExpressionBeforeDot(t *testing.T) {
	if lhs := lhsExpressionBeforeDot("  foo."); lhs != "foo" {
		t.Fatalf("got %q", lhs)
	}
	if lhs := lhsExpressionBeforeDot("a.b."); lhs != "a.b" {
		t.Fatalf("got %q", lhs)
	}
	if lhs := lhsExpressionBeforeDot("  _ = s."); lhs != "s" {
		t.Fatalf("got %q", lhs)
	}
	if lhs := lhsExpressionBeforeDot("no dot"); lhs != "" {
		t.Fatalf("got %q", lhs)
	}
}

func TestGetCompletionsForPosition_memberAfterDot(t *testing.T) {
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "dot.ft")
	const src = `package main

func f() {
  var s: String = "a"
  _ = s.len
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := "file://" + ftPath
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	// Cursor after `s.` on line `  _ = s.len`
	pos := LSPPosition{Line: 4, Character: 8}
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("analyze failed")
	}
	if ctx.ParseErr != nil {
		t.Fatalf("parse: %v", ctx.ParseErr)
	}
	if ctx.TC == nil {
		t.Fatal("no typechecker")
	}
	tokIdx := tokenIndexAtLSPPosition(ctx.Tokens, pos)
	sn := findInnermostScopeNode(ctx.Nodes, ctx.Tokens, tokIdx, ctx.TC)
	if sn == nil {
		t.Fatal("expected function scope node")
	}
	if err := ctx.TC.RestoreScope(sn); err != nil {
		t.Fatalf("RestoreScope: %v", err)
	}
	vn := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier("s")}}
	types, err := ctx.TC.InferExpressionTypeForCompletion(vn)
	if err != nil {
		t.Fatalf("infer s: %v", err)
	}
	if len(types) == 0 {
		t.Fatal("no types for s")
	}

	items := s.getCompletionsForPosition(uri, pos, &completionRequestContext{TriggerCharacter: "."})
	found := make(map[string]bool)
	for _, it := range items {
		found[it.Label] = true
	}
	if !found["len"] {
		t.Fatalf("expected builtin String.len, got %#v", found)
	}
}

func TestGetCompletionsForPosition_includesLocalVariable(t *testing.T) {
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "locals.ft")
	const src = `package main

func main() {
  var x: Int = 1
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := "file://" + ftPath
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	// After `var `, before `x` — identifier prefix empty
	pos := LSPPosition{Line: 3, Character: 6}
	items := s.getCompletionsForPosition(uri, pos, nil)
	found := make(map[string]bool)
	for _, it := range items {
		found[it.Label] = true
	}
	if !found["x"] {
		t.Fatalf("expected local x in completions, got %#v", found)
	}
}

func TestGetCompletionsForPosition_shapeFieldsAfterDot(t *testing.T) {
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "shape_dot.ft")
	const src = `package main

type Row = {
  name: String
  count: Int
}

func f() {
  var r: Row
  _ = r.name
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := "file://" + ftPath
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	// After `r.` on line `  _ = r.name`
	pos := LSPPosition{Line: 9, Character: 8}
	items := s.getCompletionsForPosition(uri, pos, &completionRequestContext{TriggerCharacter: "."})
	found := make(map[string]bool)
	for _, it := range items {
		found[it.Label] = true
	}
	if !found["name"] || !found["count"] {
		t.Fatalf("expected shape fields name and count, got %#v", found)
	}
}
