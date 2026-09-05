package lsp

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestMemberCompletionsAfterDot_fmtPackageDot(t *testing.T) {
	t.Parallel()
	const src = `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	_, uri := importTestModuleFile(t, "complete_fmt.ft", src)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil {
		t.Fatalf("analyze failed: parseErr=%v checkErr=%v", ctx.ParseErr, ctx.CheckErr)
	}
	if ctx.TC == nil || !ctx.TC.IsImportedLocalName("fmt") {
		t.Skip("fmt not loaded")
	}

	pos, found := dotPositionBeforeIdentifier(src, ctx.Tokens, "Println")
	if !found {
		t.Fatal("dot before Println not found")
	}

	items, _ := s.getCompletionsForPosition(uri, pos, &completionRequestContext{TriggerCharacter: "."})
	labels := make(map[string]bool)
	for _, it := range items {
		labels[it.Label] = true
	}

	if !labels["Println"] || !labels["Printf"] || !labels["Sprintf"] {
		t.Fatalf("expected fmt exported functions in completions, got labels: %#v", labels)
	}
}

func TestMemberCompletionsAfterDot_execPackageDot(t *testing.T) {
	t.Parallel()
	const src = `package main

import "os/exec"

func main() {
	exec.Command("ls")
}
`
	_, uri := importTestModuleFile(t, "complete_exec.ft", src)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil {
		t.Fatalf("analyze failed: parseErr=%v checkErr=%v", ctx.ParseErr, ctx.CheckErr)
	}
	if ctx.TC == nil || !ctx.TC.IsImportedLocalName("exec") {
		t.Skip("exec not loaded")
	}

	pos, found := dotPositionBeforeIdentifier(src, ctx.Tokens, "Command")
	if !found {
		t.Fatal("dot before Command not found")
	}

	items, _ := s.getCompletionsForPosition(uri, pos, &completionRequestContext{TriggerCharacter: "."})
	labels := make(map[string]bool)
	for _, it := range items {
		labels[it.Label] = true
	}

	if !labels["Command"] || !labels["Cmd"] {
		t.Fatalf("expected exec exported items (Command, Cmd) in completions, got labels: %#v", labels)
	}
}
