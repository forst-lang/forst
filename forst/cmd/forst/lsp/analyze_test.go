package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestAnalyzeForstDocument_nonFtURI(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	goURI := mustFileURI(t, filepath.Join(t.TempDir(), "x.go"))
	ctx, ok := s.analyzeForstDocument(goURI)
	if ok || ctx != nil {
		t.Fatalf("expected non-.ft URI to be rejected, ok=%v ctx=%v", ok, ctx)
	}
}

func TestAnalyzeForstDocument_openBuffer_setsTypeChecker(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module t\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ft := filepath.Join(dir, "a.ft")
	const src = "package main\n\nfunc main() {}\n"
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
		t.Fatalf("expected analyze ok, got ok=%v ctx=%v", ok, ctx)
	}
	if ctx.TC == nil {
		t.Fatal("expected typechecker on successful parse")
	}
	if ctx.FilePath != ft {
		t.Fatalf("FilePath: got %q want %q", ctx.FilePath, ft)
	}
}

func TestAnalyzeForstDocument_parseErrorStillReturnsTokens(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad.ft")
	uri := mustFileURI(t, badPath)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	// Top-level token the parser rejects (see internal/parser/parse_test.go).
	s.openDocuments[uri] = "package main\n\nunexpected\n"
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatalf("expected ctx with parse error, ok=%v", ok)
	}
	if ctx.ParseErr == nil {
		t.Fatal("expected parse error")
	}
	if len(ctx.Tokens) == 0 {
		t.Fatal("expected tokens for hover/diagnostics even when parse fails")
	}
	if ctx.TC != nil {
		t.Fatal("expected TC nil when parse fails")
	}
}
