package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

// lspPosForNthIdentToken returns the LSP position (0-based line/character) of the n-th
// identifier token with the given value in the token stream (skips EOF).
func lspPosForNthIdentToken(tokens []ast.Token, name string, n int) (LSPPosition, bool) {
	seen := 0
	for i := range tokens {
		if tokens[i].Type == ast.TokenEOF {
			continue
		}
		if tokens[i].Type != ast.TokenIdentifier || tokens[i].Value != name {
			continue
		}
		if seen == n {
			t := &tokens[i]
			return LSPPosition{Line: t.Line - 1, Character: t.Column - 1}, true
		}
		seen++
	}
	return LSPPosition{}, false
}

func TestHandleDefinition_ParameterFromBody(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "param.ft")
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

	resp := s.handleDefinition(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/definition",
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"position":     map[string]interface{}{"line": posUse.Line, "character": posUse.Character},
		}),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	loc := mustLSPLocation(t, resp.Result)
	posParam, ok := lspPosForNthIdentToken(ctx.Tokens, "x", 0)
	if !ok {
		t.Fatal("param x not found")
	}
	if loc.Range.Start.Line != posParam.Line || loc.Range.Start.Character != posParam.Character {
		t.Fatalf("want definition at param x %+v, got %+v", posParam, loc.Range.Start)
	}
}

func TestHandleDefinition_shortDeclInsideIfBlock(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "if_short.ft")
	const src = `package main

func main() {
  if true {
    z := 1
    println(string(z))
  }
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
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v parseErr=%v", ok, ctx.ParseErr)
	}
	posUse, ok := lspPosForNthIdentToken(ctx.Tokens, "z", 1)
	if !ok {
		t.Fatal("use z not found")
	}
	resp := s.handleDefinition(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/definition",
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"position":     map[string]interface{}{"line": posUse.Line, "character": posUse.Character},
		}),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	loc := mustLSPLocation(t, resp.Result)
	posDecl, ok := lspPosForNthIdentToken(ctx.Tokens, "z", 0)
	if !ok {
		t.Fatal("decl z not found")
	}
	if loc.Range.Start.Line != posDecl.Line || loc.Range.Start.Character != posDecl.Character {
		t.Fatalf("want definition at short decl %+v, got %+v", posDecl, loc.Range.Start)
	}
}

func TestDefiningTokenForLocalBinding_ifBody_matchesHandleDefinition(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "if_short_bind.ft")
	const src = `package main

func main() {
  if true {
    z := 1
    println(string(z))
  }
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
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v", ok)
	}
	posUse, ok := lspPosForNthIdentToken(ctx.Tokens, "z", 1)
	if !ok {
		t.Fatal("use z not found")
	}
	tokIdx := tokenIndexAtLSPPosition(ctx.Tokens, posUse)
	if tokIdx < 0 || tokIdx >= len(ctx.Tokens) {
		t.Fatalf("bad tokIdx %d", tokIdx)
	}
	tok := &ctx.Tokens[tokIdx]
	defTok := definingTokenForLocalBinding(ctx, tokIdx, tok)
	if defTok == nil || defTok.Value != "z" {
		t.Fatalf("definingTokenForLocalBinding: got %+v", defTok)
	}
	posDecl, ok := lspPosForNthIdentToken(ctx.Tokens, "z", 0)
	if !ok {
		t.Fatal("decl z not found")
	}
	if defTok.Line-1 != posDecl.Line || defTok.Column-1 != posDecl.Character {
		t.Fatalf("def tok at %d:%d want decl %+v", defTok.Line, defTok.Column, posDecl)
	}
}

func TestHandleReferences_ParameterAndBody_includeDeclaration(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "param_ref.ft")
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

	resp := s.handleReferences(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/references",
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"position":     map[string]interface{}{"line": posUse.Line, "character": posUse.Character},
			"context":      map[string]interface{}{"includeDeclaration": true},
		}),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	locs, ok := resp.Result.([]LSPLocation)
	if !ok {
		t.Fatalf("result type %T", resp.Result)
	}
	if len(locs) != 2 {
		t.Fatalf("want 2 refs (param + body), got %d", len(locs))
	}
}

func TestHandleReferences_shadowingDifferentBindings(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "shadow.ft")
	const src = `package main

func shadow(): Int {
  x := 1
  if true {
    x := 2
    return x
  }
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

	// Tokens named x: outer def, inner def, inner use, outer use
	posInnerUse, ok := lspPosForNthIdentToken(ctx.Tokens, "x", 2)
	if !ok {
		t.Fatal("inner use x not found")
	}
	posOuterUse, ok := lspPosForNthIdentToken(ctx.Tokens, "x", 3)
	if !ok {
		t.Fatal("outer use x not found")
	}

	respInner := s.handleReferences(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/references",
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"position":     map[string]interface{}{"line": posInnerUse.Line, "character": posInnerUse.Character},
			"context":      map[string]interface{}{"includeDeclaration": true},
		}),
	})
	if respInner.Error != nil {
		t.Fatalf("error: %+v", respInner.Error)
	}
	innerLocs, ok := respInner.Result.([]LSPLocation)
	if !ok {
		t.Fatalf("inner result type %T", respInner.Result)
	}
	if len(innerLocs) != 2 {
		t.Fatalf("inner binding: want 2 refs, got %d", len(innerLocs))
	}

	respOuter := s.handleReferences(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/references",
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"position":     map[string]interface{}{"line": posOuterUse.Line, "character": posOuterUse.Character},
			"context":      map[string]interface{}{"includeDeclaration": true},
		}),
	})
	if respOuter.Error != nil {
		t.Fatalf("error: %+v", respOuter.Error)
	}
	outerLocs, ok := respOuter.Result.([]LSPLocation)
	if !ok {
		t.Fatalf("outer result type %T", respOuter.Result)
	}
	if len(outerLocs) != 2 {
		t.Fatalf("outer binding: want 2 refs, got %d", len(outerLocs))
	}
}

func TestHandleDefinition_nonIdentifierReturnsNull(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "kw.ft")
	const src = `package main

func main() {
  return 0
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
	posReturn, ok := lspPosForNthKeyword(ctx.Tokens, ast.TokenReturn, 0)
	if !ok {
		t.Fatal("return keyword not found")
	}

	resp := s.handleDefinition(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/definition",
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"position":     map[string]interface{}{"line": posReturn.Line, "character": posReturn.Character},
		}),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	if resp.Result != nil {
		t.Fatalf("expected null definition for keyword, got %#v", resp.Result)
	}
}

func TestHandleDefinition_shortDeclInsideElseBlock(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "else_local.ft")
	const src = `package main

func main() {
  n := 2
  if n > 10 {
    println("a")
  } else {
    z := 1
    println(string(z))
  }
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
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v err=%v", ok, ctx.ParseErr)
	}
	posUse, ok := lspPosForNthIdentToken(ctx.Tokens, "z", 1)
	if !ok {
		t.Fatal("use z not found")
	}
	resp := s.handleDefinition(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/definition",
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"position":     map[string]interface{}{"line": posUse.Line, "character": posUse.Character},
		}),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	loc := mustLSPLocation(t, resp.Result)
	posDecl, ok := lspPosForNthIdentToken(ctx.Tokens, "z", 0)
	if !ok {
		t.Fatal("decl z not found")
	}
	if loc.Range.Start.Line != posDecl.Line || loc.Range.Start.Character != posDecl.Character {
		t.Fatalf("want decl %+v, got %+v", posDecl, loc.Range.Start)
	}
}

func TestHandleDefinition_ifInitShortDecl(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "if_init.ft")
	const src = `package main

func f(): Int {
  if x := 1; x > 0 {
    return x
  }
  return 0
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
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v err=%v", ok, ctx.ParseErr)
	}
	posUse, ok := lspPosForNthIdentToken(ctx.Tokens, "x", 2)
	if !ok {
		t.Fatal("return x use not found")
	}
	resp := s.handleDefinition(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/definition",
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"position":     map[string]interface{}{"line": posUse.Line, "character": posUse.Character},
		}),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	loc := mustLSPLocation(t, resp.Result)
	posDecl, ok := lspPosForNthIdentToken(ctx.Tokens, "x", 0)
	if !ok {
		t.Fatal("x in init not found")
	}
	if loc.Range.Start.Line != posDecl.Line || loc.Range.Start.Character != posDecl.Character {
		t.Fatalf("want decl %+v, got %+v", posDecl, loc.Range.Start)
	}
}

func TestHandleDefinition_ifBlockShortDecl(t *testing.T) {
	t.Parallel()
	server := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	filePath := filepath.Join(dir, "if_local.ft")
	const source = `package main

func f(): Int {
  if true {
    local := 1
    return local
  }
  return 0
}
`
	if err := os.WriteFile(filePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, filePath)
	server.documentMu.Lock()
	server.openDocuments[uri] = source
	server.documentMu.Unlock()

	context, ok := server.analyzeForstDocument(uri)
	if !ok || context == nil || context.ParseErr != nil {
		t.Fatalf("analyze: ok=%v parseErr=%v", ok, context.ParseErr)
	}
	usePos, ok := lspPosForNthIdentToken(context.Tokens, "local", 1)
	if !ok {
		t.Fatal("local use token not found")
	}

	response := server.handleDefinition(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/definition",
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"position":     map[string]interface{}{"line": usePos.Line, "character": usePos.Character},
		}),
	})
	if response.Error != nil {
		t.Fatalf("definition error: %+v", response.Error)
	}
	location := mustLSPLocation(t, response.Result)
	defPos, ok := lspPosForNthIdentToken(context.Tokens, "local", 0)
	if !ok {
		t.Fatal("local definition token not found")
	}
	if location.Range.Start.Line != defPos.Line || location.Range.Start.Character != defPos.Character {
		t.Fatalf("expected local definition at %+v, got %+v", defPos, location.Range.Start)
	}
}

func TestHandleReferences_ifBlockShortDecl(t *testing.T) {
	t.Parallel()
	server := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	filePath := filepath.Join(dir, "if_refs.ft")
	const source = `package main

func f(): Int {
  if true {
    local := 1
    return local
  }
  return 0
}
`
	if err := os.WriteFile(filePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, filePath)
	server.documentMu.Lock()
	server.openDocuments[uri] = source
	server.documentMu.Unlock()

	context, ok := server.analyzeForstDocument(uri)
	if !ok || context == nil || context.ParseErr != nil {
		t.Fatalf("analyze: ok=%v parseErr=%v", ok, context.ParseErr)
	}
	usePos, ok := lspPosForNthIdentToken(context.Tokens, "local", 1)
	if !ok {
		t.Fatal("local use token not found")
	}

	response := server.handleReferences(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/references",
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"position":     map[string]interface{}{"line": usePos.Line, "character": usePos.Character},
			"context":      map[string]interface{}{"includeDeclaration": true},
		}),
	})
	if response.Error != nil {
		t.Fatalf("references error: %+v", response.Error)
	}
	locations, ok := response.Result.([]LSPLocation)
	if !ok {
		t.Fatalf("unexpected references result type: %T", response.Result)
	}
	if len(locations) != 2 {
		t.Fatalf("expected 2 refs for if-block local binding, got %d", len(locations))
	}
}

func lspPosForNthKeyword(tokens []ast.Token, typ ast.TokenIdent, n int) (LSPPosition, bool) {
	seen := 0
	for i := range tokens {
		if tokens[i].Type == ast.TokenEOF {
			continue
		}
		if tokens[i].Type != typ {
			continue
		}
		if seen == n {
			t := &tokens[i]
			return LSPPosition{Line: t.Line - 1, Character: t.Column - 1}, true
		}
		seen++
	}
	return LSPPosition{}, false
}
