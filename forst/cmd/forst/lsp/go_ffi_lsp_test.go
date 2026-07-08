package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

func dotPositionBeforeIdentifier(content string, tokens []ast.Token, id string) (LSPPosition, bool) {
	for i := range tokens {
		if tokens[i].Type != ast.TokenIdentifier || tokens[i].Value != id {
			continue
		}
		if i == 0 || tokens[i-1].Type != ast.TokenDot {
			continue
		}
		return lspPositionAfterDot(content, tokens[i-1]), true
	}
	return LSPPosition{}, false
}

func assertNoGoImportDiagnostics(t *testing.T, diags []LSPDiagnostic) {
	t.Helper()
	for _, d := range diags {
		if d.Code == "go-import" || strings.Contains(d.Message, "types not loaded") {
			t.Fatalf("unexpected go-import diagnostic: code=%q msg=%q", d.Code, d.Message)
		}
	}
}

func TestCompileForstFile_nestedProbeExecFt_noGoImportDiagnostic(t *testing.T) {
	t.Parallel()
	_, ftPath := testutil.WriteProbeModuleFixture(t, false)
	src, err := os.ReadFile(ftPath)
	if err != nil {
		t.Fatal(err)
	}
	s := NewLSPServer("8080", logrus.New())
	diags := s.compileForstFile(ftPath, string(src), nil)
	assertNoGoImportDiagnostics(t, diags)
}

func TestProcessForstFile_nestedProbeExecFt_noGoImportDiagnostic(t *testing.T) {
	t.Parallel()
	_, ftPath := testutil.WriteProbeModuleFixture(t, false)
	src, err := os.ReadFile(ftPath)
	if err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = string(src)
	s.documentMu.Unlock()

	diags := s.processForstFile(uri, string(src))
	assertNoGoImportDiagnostics(t, diags)
}

func TestProcessForstFile_nestedProbeExecAndWrite_noGoImportOnExec(t *testing.T) {
	t.Parallel()
	root, execPath := testutil.WriteProbeModuleFixture(t, true)
	writePath := filepath.Join(root, "internal", "probe", "write.ft")
	execSrc, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatal(err)
	}
	writeSrc, err := os.ReadFile(writePath)
	if err != nil {
		t.Fatal(err)
	}
	execURI := mustFileURI(t, execPath)
	writeURI := mustFileURI(t, writePath)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[execURI] = string(execSrc)
	s.openDocuments[writeURI] = string(writeSrc)
	s.documentMu.Unlock()

	diags := s.processForstFile(execURI, string(execSrc))
	assertNoGoImportDiagnostics(t, diags)
}

func TestAnalyzeForstDocument_nestedProbeExecFt(t *testing.T) {
	t.Parallel()
	_, ftPath := testutil.WriteProbeModuleFixture(t, false)
	src, err := os.ReadFile(ftPath)
	if err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = string(src)
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("analyze failed")
	}
	if ctx.ParseErr != nil {
		t.Fatalf("parse: %v", ctx.ParseErr)
	}
	if ctx.CheckErr != nil {
		t.Fatalf("check: %v", ctx.CheckErr)
	}
	if ctx.TC == nil || !ctx.TC.GoImportPackageLoaded("exec") {
		t.Fatal("exec package not loaded in LSP analyze")
	}
}

func TestFindHoverForPosition_execCommand(t *testing.T) {
	t.Parallel()
	const src = `package main

import "os/exec"

func main() {
	exec.Command("true")
}
`
	_, uri := importTestModuleFile(t, "hover_exec.ft", src)
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

func TestFindHoverForPosition_cmdProcessStateExitCode(t *testing.T) {
	t.Parallel()
	const src = `package main

import "os/exec"

func main() {
	cmd := exec.Command("true")
	_ = cmd.ProcessState.ExitCode()
}
`
	_, uri := importTestModuleFile(t, "hover_exitcode.ft", src)
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

	var exitTok *ast.Token
	for i := range ctx.Tokens {
		tok := &ctx.Tokens[i]
		if tok.Type == ast.TokenIdentifier && tok.Value == "ExitCode" {
			exitTok = tok
			break
		}
	}
	if exitTok == nil {
		t.Fatal("ExitCode token not found")
	}
	h := s.findHoverForPosition(uri, LSPPosition{Line: exitTok.Line - 1, Character: exitTok.Column - 1})
	if h == nil || !strings.Contains(h.Contents.Value, "ExitCode") {
		t.Fatalf("expected ExitCode hover, got %v", h)
	}
}

func TestMemberCompletionsAfterDot_cmdDot(t *testing.T) {
	t.Parallel()
	const src = `package main

import "os/exec"

func main() {
	cmd := exec.Command("true")
	cmd.Run()
}
`
	_, uri := importTestModuleFile(t, "complete_cmd.ft", src)
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

	pos, found := dotPositionBeforeIdentifier(src, ctx.Tokens, "Run")
	if !found {
		t.Fatal("dot before Run not found")
	}
	items, _ := s.getCompletionsForPosition(uri, pos, &completionRequestContext{TriggerCharacter: "."})
	labels := make(map[string]bool)
	for _, it := range items {
		labels[it.Label] = true
	}
	if !labels["Run"] && !labels["ProcessState"] {
		t.Fatalf("expected cmd Go members, got %#v", labels)
	}
}

func TestMemberCompletionsAfterDot_cmdDot_direct(t *testing.T) {
	t.Parallel()
	const src = `package main

import "os/exec"

func main() {
	cmd := exec.Command("true")
	cmd.Run()
}
`
	_, uri := importTestModuleFile(t, "complete_cmd_direct.ft", src)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		t.Fatalf("analyze failed")
	}
	if !ctx.TC.GoImportPackageLoaded("exec") {
		t.Skip("os/exec not loaded")
	}
	pos, found := dotPositionBeforeIdentifier(src, ctx.Tokens, "Run")
	if !found {
		t.Fatal("dot before Run not found")
	}
	scopeNode := findInnermostScopeNode(ctx.Nodes, ctx.Tokens, tokenIndexAtLSPPosition(ctx.Tokens, pos), ctx.TC)
	if scopeNode != nil {
		_ = ctx.TC.RestoreScope(scopeNode)
	}
	items := memberCompletionsAfterDot(ctx, pos, "")
	labels := make(map[string]bool)
	for _, it := range items {
		labels[it.Label] = true
	}
	if !labels["Run"] && !labels["ProcessState"] {
		t.Fatalf("expected cmd Go members from memberCompletionsAfterDot, got %#v", labels)
	}
}

func TestMemberCompletionsAfterDot_cmdProcessStateDot(t *testing.T) {
	t.Parallel()
	const src = `package main

import "os/exec"

func main() {
	cmd := exec.Command("true")
	cmd.ProcessState.ExitCode()
}
`
	_, uri := importTestModuleFile(t, "complete_ps.ft", src)
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

	pos, found := dotPositionBeforeIdentifier(src, ctx.Tokens, "ExitCode")
	if !found {
		t.Fatal("dot before ExitCode not found")
	}
	items, _ := s.getCompletionsForPosition(uri, pos, &completionRequestContext{TriggerCharacter: "."})
	labels := make(map[string]bool)
	for _, it := range items {
		labels[it.Label] = true
	}
	if !labels["ExitCode"] {
		t.Fatalf("expected ExitCode on ProcessState, got %#v", labels)
	}
}

func TestMemberCompletionsAfterDot_execCommandCallResult(t *testing.T) {
	t.Parallel()
	const src = `package main

import "os/exec"

func main() {
	exec.Command("true").Run()
}
`
	_, uri := importTestModuleFile(t, "complete_call.ft", src)
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

	pos, found := dotPositionBeforeIdentifier(src, ctx.Tokens, "Run")
	if !found {
		t.Fatal("dot before Run not found")
	}
	items, _ := s.getCompletionsForPosition(uri, pos, &completionRequestContext{TriggerCharacter: "."})
	labels := make(map[string]bool)
	for _, it := range items {
		labels[it.Label] = true
	}
	if !labels["Run"] {
		t.Fatalf("expected Run on exec.Command result, got %#v", labels)
	}
}
