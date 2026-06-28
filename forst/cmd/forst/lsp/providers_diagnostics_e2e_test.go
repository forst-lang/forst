package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestAnalyzeForstDocument_providersUnsatisfiedDiagnostic(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module providers_diag\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ft := filepath.Join(dir, "needs.ft")
	src := `package main

import "testing"

type Logger = { info(msg String) }

func needsLogger() {
	use logger: Logger
}

func TestX(t *testing.T) {
	needsLogger()
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("analyze failed")
	}
	if ctx.CheckErr == nil {
		t.Fatal("expected typecheck error")
	}
	diag := diagnosticForTypecheckError(uri, src, ctx.CheckErr, "forst-typechecker", ErrorCodeTypeMismatch)
	if diag.Code != "providers-unsatisfied" {
		t.Fatalf("code = %q, want providers-unsatisfied", diag.Code)
	}
	if !strings.Contains(diag.Message, "Logger") {
		t.Fatalf("message = %q", diag.Message)
	}
	if !strings.Contains(diag.Message, "TestX → needsLogger") {
		t.Fatalf("expected obligation chain in message: %q", diag.Message)
	}
}
