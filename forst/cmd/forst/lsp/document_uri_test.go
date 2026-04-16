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
	if want := fileURIForLocalPath(abs); canonicalFileURI(uri) != want {
		t.Fatalf("canonicalFileURI: got %q want %q", canonicalFileURI(uri), want)
	}
}

func TestFilePathFromDocumentURI_percentEncoded(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sub := filepath.Join(dir, "name with space.ft")
	if err := os.WriteFile(sub, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(sub)
	if err != nil {
		t.Fatal(err)
	}
	encoded := strings.ReplaceAll(fileURIForLocalPath(abs), " ", "%20")
	got := filePathFromDocumentURI(encoded)
	if filepath.Clean(got) != filepath.Clean(abs) {
		t.Fatalf("percent-encoded segment: got %q want %q", got, abs)
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

func TestLegacyLocalPathFromFileURI_fileSchemePath(t *testing.T) {
	t.Parallel()
	p, ok := legacyLocalPathFromFileURI("file:///tmp/example.ft")
	if !ok || p == "" {
		t.Fatalf("got ok=%v p=%q", ok, p)
	}
	if !strings.HasSuffix(filepath.ToSlash(p), "/tmp/example.ft") {
		t.Fatalf("unexpected path %q", p)
	}
}

func TestLegacyLocalPathFromFileURI_noSchemePrefix(t *testing.T) {
	t.Parallel()
	if _, ok := legacyLocalPathFromFileURI("https://x/y"); ok {
		t.Fatal("expected false without file:// prefix")
	}
}

func TestLocalPathFromFileURI_nonFileSchemeFallsBackToLegacy(t *testing.T) {
	t.Parallel()
	// url.Parse succeeds but scheme != file → legacy (no file:// prefix → false).
	if _, ok := localPathFromFileURI("http://example.com/x.ft"); ok {
		t.Fatal("expected non-file scheme to fail")
	}
}
