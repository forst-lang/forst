package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

// Tests findInnermostScopeNode / deepestScopeInFunction via the same lex+parse+typecheck path as the LSP server.
func TestFindInnermostScopeNode_cursorInParamList(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("0", log)
	dir := t.TempDir()
	p := filepath.Join(dir, "params.ft")
	const src = `package main

func f(a Int, b Int) {
  return
}
`
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, p)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v err=%v", ok, ctx)
	}
	// Cursor on first parameter name "a"
	pos := LSPPosition{Line: 3, Character: 6}
	tokIdx := tokenIndexAtLSPPosition(ctx.Tokens, pos)
	if n := findInnermostScopeNode(ctx.Nodes, ctx.Tokens, tokIdx, ctx.TC); n == nil {
		t.Fatal("expected scope node in param list")
	}
}

func TestFindInnermostScopeNode_cursorInFunctionBody(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("0", log)
	dir := t.TempDir()
	p := filepath.Join(dir, "body.ft")
	const src = `package main

func main() {
  x := 1
  println(x)
}
`
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, p)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v", ok)
	}
	pos := LSPPosition{Line: 4, Character: 3}
	tokIdx := tokenIndexAtLSPPosition(ctx.Tokens, pos)
	if n := findInnermostScopeNode(ctx.Nodes, ctx.Tokens, tokIdx, ctx.TC); n == nil {
		t.Fatal("expected inner scope")
	}
}

func TestFindInnermostScopeNode_typeGuardFile(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("0", log)
	dir := t.TempDir()
	p := filepath.Join(dir, "tg.ft")
	const src = `package main

type P = String

is (p P) Strong {
  ensure p is Min(3)
}
`
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, p)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v", ok)
	}
	pos := LSPPosition{Line: 5, Character: 4}
	tokIdx := tokenIndexAtLSPPosition(ctx.Tokens, pos)
	if n := findInnermostScopeNode(ctx.Nodes, ctx.Tokens, tokIdx, ctx.TC); n == nil {
		t.Fatal("expected type-guard scope")
	}
}

func TestFindInnermostScopeNode_elseIfBranch(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("0", log)
	dir := t.TempDir()
	p := filepath.Join(dir, "elseif.ft")
	const src = `package main

func main() {
  n := 2
  if n > 10 {
    println("a")
  } else if n < 0 {
    println("b")
  } else {
    println("c")
  }
}
`
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, p)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v err=%v", ok, ctx)
	}
	// Inside else-if body: token for "println" on the line with "b"
	pos := LSPPosition{Line: 7, Character: 4}
	tokIdx := tokenIndexAtLSPPosition(ctx.Tokens, pos)
	if n := findInnermostScopeNode(ctx.Nodes, ctx.Tokens, tokIdx, ctx.TC); n == nil {
		t.Fatal("expected scope in else-if branch")
	}
}

func TestListAtPosition_elseIfBranch(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("0", log)
	dir := t.TempDir()
	p := filepath.Join(dir, "elseif_cpl.ft")
	const src = `package main

func main() {
  n := 2
  if n > 10 {
    println("a")
  } else if n < 0 {
    println("b")
  } else {
    println("c")
  }
}
`
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, p)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()
	pos := LSPPosition{Line: 7, Character: 4}
	items, _ := s.getCompletionsForPosition(uri, pos, nil)
	if len(items) == 0 {
		t.Fatal("expected list items")
	}
}

func TestFindInnermostScopeNode_finalElseBranch(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("0", log)
	dir := t.TempDir()
	p := filepath.Join(dir, "else_final.ft")
	const src = `package main

func main() {
  n := 2
  if n > 10 {
    println("a")
  } else if n < 0 {
    println("b")
  } else {
    println("c")
  }
}
`
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, p)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v err=%v", ok, ctx)
	}
	pos := LSPPosition{Line: 9, Character: 4}
	tokIdx := tokenIndexAtLSPPosition(ctx.Tokens, pos)
	if n := findInnermostScopeNode(ctx.Nodes, ctx.Tokens, tokIdx, ctx.TC); n == nil {
		t.Fatal("expected scope in final else branch")
	}
}

func TestFindInnermostScopeNode_ensureBlockBody(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("0", log)
	dir := t.TempDir()
	p := filepath.Join(dir, "ensure_block.ft")
	const src = `package main

func main() {
  x := 1
  ensure x is Min(1) {
    println(string(x))
  }
}
`
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, p)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v err=%v", ok, ctx)
	}
	pos := LSPPosition{Line: 5, Character: 4}
	tokIdx := tokenIndexAtLSPPosition(ctx.Tokens, pos)
	if n := findInnermostScopeNode(ctx.Nodes, ctx.Tokens, tokIdx, ctx.TC); n == nil {
		t.Fatal("expected scope inside ensure block")
	}
}

func TestCrossBufferTopLevelItems_twoOpenBuffers(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("0", log)
	dir := t.TempDir()
	a := filepath.Join(dir, "x.ft")
	b := filepath.Join(dir, "y.ft")
	const srcA = `package app

func FromA(): Int { return 1 }
`
	const srcB = `package app

func main() {
}
`
	if err := os.WriteFile(a, []byte(srcA), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(srcB), 0o644); err != nil {
		t.Fatal(err)
	}
	uriA := mustFileURI(t, a)
	uriB := mustFileURI(t, b)
	s.documentMu.Lock()
	s.openDocuments[uriA] = srcA
	s.openDocuments[uriB] = srcB
	s.documentMu.Unlock()

	items := s.crossBufferTopLevelCompletionItems(uriB, "app", "")
	if len(items) == 0 {
		t.Fatal("expected cross-buffer symbols")
	}
	seen := false
	for _, it := range items {
		if it.Label == "FromA" {
			seen = true
			break
		}
	}
	if !seen {
		t.Fatalf("expected FromA from peer buffer, got %#v", items)
	}
}
