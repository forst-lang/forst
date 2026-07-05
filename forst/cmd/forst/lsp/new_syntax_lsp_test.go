package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestFindHoverForPosition_postfixMethodOnQualifiedCall(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "postfix_hover.ft")
	const src = `package main

import "time"

func main() {
	_ = time.Now().Format("2006-01-02")
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := fileURIForLocalPath(ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze: ok=%v parseErr=%v checkErr=%v", ok, ctx.ParseErr, ctx.CheckErr)
	}
	if !ctx.TC.GoImportPackageLoaded("time") {
		t.Skip("time package not loaded")
	}

	var formatTok *ast.Token
	for i := range ctx.Tokens {
		tok := &ctx.Tokens[i]
		if tok.Type == ast.TokenIdentifier && tok.Value == "Format" {
			if i >= 2 && ctx.Tokens[i-1].Type == ast.TokenDot {
				formatTok = tok
				break
			}
		}
	}
	if formatTok == nil {
		t.Fatal("Format token not found")
	}
	h := s.findHoverForPosition(uri, LSPPosition{Line: formatTok.Line - 1, Character: formatTok.Column - 1})
	if h == nil || !strings.Contains(h.Contents.Value, "Format") {
		t.Fatalf("expected Format method hover, got %v", h)
	}
}

func TestDocumentSymbol_packageLevelVar(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "pkg_var.ft")
	const src = `package main

var Version = "1.0"

func main() {}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := fileURIForLocalPath(ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil {
		t.Fatalf("analyze failed: parseErr=%v", ctx.ParseErr)
	}
	syms := symbolsFromParsedDocument(uri, ctx.Tokens, ctx.Nodes)
	var found bool
	for _, sym := range syms {
		if sym.Name == "Version" && sym.Kind == lspSymbolKindVariable {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected package var Version in symbols, got %v", syms)
	}
}

func TestFindDefinitionForPosition_packageLevelVar(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "pkg_var_def.ft")
	const src = `package main

var Version = "1.0"

func main() {
	println(Version)
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := fileURIForLocalPath(ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze failed")
	}
	var useTok *ast.Token
	for i := range ctx.Tokens {
		tok := &ctx.Tokens[i]
		if tok.Type == ast.TokenIdentifier && tok.Value == "Version" {
			if i > 0 && ctx.Tokens[i-1].Type == ast.TokenVar {
				continue
			}
			useTok = tok
		}
	}
	if useTok == nil {
		t.Fatal("usage token not found")
	}
	loc := s.findDefinitionForPosition(uri, LSPPosition{Line: useTok.Line - 1, Character: useTok.Column - 1})
	if loc == nil {
		t.Fatal("expected definition location for package var")
	}
	defTok := findPackageVarNameToken(ctx.Tokens, "Version")
	if defTok == nil {
		t.Fatal("def token not found")
	}
	if loc.Range.Start.Line != defTok.Line-1 {
		t.Fatalf("def line = %d want %d", loc.Range.Start.Line, defTok.Line-1)
	}
}
