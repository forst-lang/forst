package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestIotaConst_hoverDefinitionSymbols(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "iota.ft")
	const src = `package main

const (
  A = iota
  B
)

func main() {
  println(A)
  println(B)
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("0", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v ctx=%v", ok, ctx)
	}
	if !ctx.TC.IsTopLevelPackageConst("A") || !ctx.TC.IsTopLevelPackageConst("B") {
		t.Fatal("expected A and B registered as package consts")
	}

	var aUse *ast.Token
	for i := range ctx.Tokens {
		tok := &ctx.Tokens[i]
		if tok.Type == ast.TokenIdentifier && tok.Value == "A" && tok.Line >= 8 {
			aUse = tok
			break
		}
	}
	if aUse == nil {
		t.Fatal("A use site not found")
	}
	pos := LSPPosition{Line: aUse.Line - 1, Character: aUse.Column - 1}
	h := s.findHoverForPosition(uri, pos)
	if h == nil || h.Contents.Value == "" {
		t.Fatalf("expected hover for A, got %#v", h)
	}
	def := s.findDefinitionForPosition(uri, pos)
	if def == nil {
		t.Fatal("expected definition for A")
	}
	if def.Range.Start.Line != 3 { // `A = iota` is line 4 → 0-based 3
		t.Fatalf("definition line = %d, want 3; loc=%+v", def.Range.Start.Line, def)
	}

	syms := symbolsFromParsedDocument(uri, ctx.Tokens, ctx.Nodes)
	found := map[string]int{}
	for _, sym := range syms {
		found[sym.Name] = sym.Kind
	}
	if found["A"] != lspSymbolKindConstant || found["B"] != lspSymbolKindConstant {
		t.Fatalf("const symbols missing: %#v", found)
	}
}

func TestFuncLit_completionOffersParam(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "funclit.ft")
	const src = `package main

func main() {
  f := func(x Int): Int {
    return x
  }
  _ = f
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("0", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v", ok)
	}
	// Cursor on `x` in `return x`
	var xTok *ast.Token
	for i := range ctx.Tokens {
		tok := &ctx.Tokens[i]
		if tok.Type == ast.TokenIdentifier && tok.Value == "x" && tok.Line >= 5 {
			xTok = tok
			break
		}
	}
	if xTok == nil {
		t.Fatal("x token not found")
	}
	pos := LSPPosition{Line: xTok.Line - 1, Character: xTok.Column - 1}
	tokIdx := tokenIndexAtLSPPosition(ctx.Tokens, pos)
	scopeNode := findInnermostScopeNode(ctx.Nodes, ctx.Tokens, tokIdx, ctx.TC)
	if scopeNode == nil {
		t.Fatal("expected FuncLit scope")
	}
	switch scopeNode.(type) {
	case ast.FunctionLiteralNode, *ast.FunctionLiteralNode:
	default:
		t.Fatalf("scope node type %T, want FunctionLiteralNode", scopeNode)
	}
	_ = ctx.TC.RestoreScope(scopeNode)
	visible := ctx.TC.VisibleVariableLikeSymbols()
	foundX := false
	for _, id := range visible {
		if id == "x" {
			foundX = true
			break
		}
	}
	if !foundX {
		t.Fatalf("expected x in FuncLit scope, got %v", visible)
	}

	items, _ := s.getCompletionsForPosition(uri, pos, nil)
	foundItem := false
	for _, it := range items {
		if it.Label == "x" {
			foundItem = true
			break
		}
	}
	if !foundItem {
		t.Fatalf("completion missing x: %#v", items)
	}
}

func TestEmbed_memberCompletionPromotesFields(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "embed.ft")
	const srcOK = `package main

type Base = {
  Name: String
}

type Outer = {
  Base
}

func main() {
  o := Outer{ Base: { Name: "n" } }
  _ = o.Name
}
`
	if err := os.WriteFile(ft, []byte(srcOK), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("0", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = srcOK
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v err=%v", ok, ctx)
	}
	names := ctx.TC.ListFieldNamesForType(ast.TypeNode{Ident: "Outer"})
	foundName, foundBase := false, false
	for _, n := range names {
		if n == "Name" {
			foundName = true
		}
		if n == "Base" {
			foundBase = true
		}
	}
	if !foundName || !foundBase {
		t.Fatalf("expected Base and promoted Name, got %v", names)
	}
}

func TestFieldHover_includesStructTag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "tag.ft")
	const src = `package main

type Row = {
  Id: Int ` + "`json:\"id\"`" + `
}

func main() {
  r := Row{ Id: 1 }
  _ = r.Id
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("0", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v parse=%v check=%v", ok, ctx.ParseErr, ctx.CheckErr)
	}
	f, ok := ctx.TC.ShapeFieldFromTypeDef("Row", "Id")
	if !ok || f.Tag != `json:"id"` {
		t.Fatalf("ShapeField tag = %#v ok=%v", f, ok)
	}
	md, _, ok := ctx.TC.FieldHoverMarkdown("r", ast.SourceSpan{}, []string{"Id"}, ast.SourceSpan{})
	if !ok || !strings.Contains(md, `json:"id"`) {
		t.Fatalf("FieldHoverMarkdown = %q ok=%v", md, ok)
	}
	var idTok *ast.Token
	for i := range ctx.Tokens {
		tok := &ctx.Tokens[i]
		if tok.Type == ast.TokenIdentifier && tok.Value == "Id" && i >= 2 && ctx.Tokens[i-1].Type == ast.TokenDot {
			idTok = tok
			break
		}
	}
	if idTok == nil {
		t.Fatal("Id use not found")
	}
	h := s.findHoverForPosition(uri, LSPPosition{Line: idTok.Line - 1, Character: idTok.Column - 1})
	if h == nil || !strings.Contains(h.Contents.Value, `json:"id"`) {
		t.Fatalf("expected tag in hover, got %#v", h)
	}
}
