package lsp

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/goload"

	"github.com/sirupsen/logrus"
)

func TestFindHoverForPosition_samePackageGoFunc_includesGodoc(t *testing.T) {
	t.Parallel()
	goload.ClearLoadCacheForTest()
	_, uri := writeSamePackageGoLSPFixture(t)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = samePkgGoFtSource
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil {
		t.Fatalf("analyze failed: parseErr=%v checkErr=%v", ctx.ParseErr, ctx.CheckErr)
	}
	if ctx.TC == nil || !ctx.TC.SamePackageGoLoaded() {
		t.Fatal("expected same-package Go loaded")
	}

	var labelTok *ast.Token
	for i := range ctx.Tokens {
		tok := &ctx.Tokens[i]
		if tok.Type == ast.TokenIdentifier && tok.Value == "StringFromBytes" {
			labelTok = tok
			break
		}
	}
	if labelTok == nil {
		t.Fatal("StringFromBytes token not found")
	}
	h := s.findHoverForPosition(uri, LSPPosition{Line: labelTok.Line - 1, Character: labelTok.Column - 1})
	if h == nil || h.Contents.Value == "" {
		t.Fatal("expected hover for StringFromBytes")
	}
	if !strings.Contains(h.Contents.Value, "StringFromBytes") {
		t.Fatalf("hover should mention StringFromBytes: %q", h.Contents.Value)
	}
	if !strings.Contains(h.Contents.Value, "memo file") {
		t.Fatalf("hover should include godoc: %q", h.Contents.Value)
	}
}

func TestFindHoverForPosition_cmdRun_includesSignatureOrGodoc(t *testing.T) {
	t.Parallel()
	const src = `package main

import "os/exec"

func main() {
	cmd := exec.Command("true")
	cmd.Run()
}
`
	_, uri := importTestModuleFile(t, "hover_cmd_run.ft", src)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil {
		t.Fatalf("analyze failed: parseErr=%v checkErr=%v", ctx.ParseErr, ctx.CheckErr)
	}
	if ctx.TC == nil || !ctx.TC.GoImportPackageLoaded("exec") {
		t.Skip("os/exec not loaded")
	}

	var runTok *ast.Token
	for i := range ctx.Tokens {
		tok := &ctx.Tokens[i]
		if tok.Type == ast.TokenIdentifier && tok.Value == "Run" {
			runTok = tok
			break
		}
	}
	if runTok == nil {
		t.Fatal("Run token not found")
	}
	h := s.findHoverForPosition(uri, LSPPosition{Line: runTok.Line - 1, Character: runTok.Column - 1})
	if h == nil || h.Contents.Value == "" {
		t.Fatal("expected hover for cmd.Run")
	}
	if !strings.Contains(h.Contents.Value, "Run") {
		t.Fatalf("hover should mention Run: %q", h.Contents.Value)
	}
}

func TestFindHoverForPosition_execCommand_includesGodoc(t *testing.T) {
	t.Parallel()
	const src = `package main

import "os/exec"

func main() {
	exec.Command("true")
}
`
	_, uri := importTestModuleFile(t, "hover_exec_command_godoc.ft", src)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil {
		t.Fatalf("analyze failed")
	}
	if ctx.TC == nil || !ctx.TC.GoImportPackageLoaded("exec") {
		t.Skip("os/exec not loaded")
	}

	var cmdTok *ast.Token
	for i := range ctx.Tokens {
		tok := &ctx.Tokens[i]
		if tok.Type == ast.TokenIdentifier && tok.Value == "Command" {
			cmdTok = tok
			break
		}
	}
	if cmdTok == nil {
		t.Fatal("Command token not found")
	}
	h := s.findHoverForPosition(uri, LSPPosition{Line: cmdTok.Line - 1, Character: cmdTok.Column - 1})
	if h == nil || !strings.Contains(h.Contents.Value, "Command") {
		t.Fatalf("expected Command hover, got %v", h)
	}
}

func TestFindHoverForPosition_goInteropGreetUpper_includesGodoc(t *testing.T) {
	t.Parallel()
	ftPath, _, src := writeGoInteropLSPFixture(t)
	s := NewLSPServer("8080", logrus.New())
	uri := mustFileURI(t, ftPath)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil {
		t.Fatalf("analyze failed: parseErr=%v checkErr=%v", ctx.ParseErr, ctx.CheckErr)
	}
	if ctx.TC == nil || !ctx.TC.SamePackageGoLoaded() {
		t.Fatal("expected same-package Go loaded")
	}

	pos := lspPositionOfIdentifier(src, "GreetUpper")
	h := s.findHoverForPosition(uri, pos)
	if h == nil || h.Contents.Value == "" {
		t.Fatal("expected hover for GreetUpper")
	}
	if !strings.Contains(h.Contents.Value, "GreetUpper") {
		t.Fatalf("hover should mention GreetUpper: %q", h.Contents.Value)
	}
	if !strings.Contains(h.Contents.Value, "hand-written Go") {
		t.Fatalf("hover should include godoc from helpers.go: %q", h.Contents.Value)
	}
}
