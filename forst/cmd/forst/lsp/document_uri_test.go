package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilePathFromDocumentURI_removesScheme(t *testing.T) {
	t.Parallel()
	got := filePathFromDocumentURI("file:///tmp/t.ft")
	if strings.Contains(got, "file:") {
		t.Fatalf("scheme left in %q", got)
	}
	if !strings.HasSuffix(got, "t.ft") {
		t.Fatalf("unexpected path %q", got)
	}
}

func TestFileURIForLocalPath_roundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "round.ft")
	if err := os.WriteFile(p, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	uri := fileURIForLocalPath(abs)
	back := filePathFromDocumentURI(uri)
	if filepath.Clean(back) != filepath.Clean(abs) {
		t.Fatalf("round trip: got %q want %q", back, abs)
	}
}

func TestIsForstDocumentURI(t *testing.T) {
	t.Parallel()
	if !isForstDocumentURI("file:///x.ft") {
		t.Fatal("expected .ft file URI")
	}
	if isForstDocumentURI("file:///x.go") {
		t.Fatal("expected non-.ft to be false")
	}
	if isForstDocumentURI("http://x.ft") {
		t.Fatal("expected non-file scheme false")
	}
}
