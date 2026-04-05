package lsp

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestMergeSamePackageDiskFt_includesUnopenedPeerFt(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)

	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.ft")
	bPath := filepath.Join(dir, "b.ft")
	const srcA = `package main

func foo(): Int {
	return 1
}
`
	const srcB = `package main

func bar(): Int {
	return 2
}
`
	if err := os.WriteFile(aPath, []byte(srcA), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bPath, []byte(srcB), 0o644); err != nil {
		t.Fatal(err)
	}

	uriA := mustFileURI(t, aPath)

	s.documentMu.Lock()
	s.openDocuments[uriA] = srcA
	s.documentMu.Unlock()

	merged := s.mergeSamePackageDiskFt(dir, "main", []string{uriA})
	if len(merged) != 2 {
		t.Fatalf("want 2 URIs (open a + disk b), got %d: %v", len(merged), merged)
	}
	uriB := mustFileURI(t, bPath)
	want := []string{uriA, uriB}
	slices.Sort(want)
	got := append([]string(nil), merged...)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("mergeSamePackageDiskFt = %v, want %v", merged, want)
	}
}

func TestPackageGroupFingerprintFromContents_orderSensitive(t *testing.T) {
	t.Parallel()
	uris := []string{"file:///a.ft", "file:///b.ft"}
	c1 := map[string]string{
		"file:///a.ft": "x",
		"file:///b.ft": "y",
	}
	c2 := map[string]string{
		"file:///a.ft": "x",
		"file:///b.ft": "z",
	}
	fp1 := packageGroupFingerprintFromContents(uris, c1)
	fp2 := packageGroupFingerprintFromContents(uris, c2)
	if fp1 == fp2 {
		t.Fatal("expected different fingerprints when buffer content changes")
	}

	uris2 := []string{"file:///b.ft", "file:///a.ft"}
	fpReorder := packageGroupFingerprintFromContents(uris2, c1)
	if fp1 == fpReorder {
		t.Fatal("packageGroupFingerprintFromContents: expected different fingerprints when URI order changes (same contents map)")
	}
}

func TestSamePackageOpenURIs_includesDiskPeerWhenBufferOpen(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)

	dir := t.TempDir()
	aPath := filepath.Join(dir, "open.ft")
	bPath := filepath.Join(dir, "peer.ft")
	const srcA = `package main

func fromOpen(): Int {
	return 1
}
`
	const srcB = `package main

func fromPeer(): Int {
	return 2
}
`
	if err := os.WriteFile(aPath, []byte(srcA), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bPath, []byte(srcB), 0o644); err != nil {
		t.Fatal(err)
	}

	uriA := mustFileURI(t, aPath)
	uriB := mustFileURI(t, bPath)

	s.documentMu.Lock()
	s.openDocuments[uriA] = srcA
	s.documentMu.Unlock()

	out := s.samePackageOpenURIs(uriA)
	if len(out) < 2 {
		t.Fatalf("expected at least open + disk peer, got %v", out)
	}
	var hasB bool
	for _, u := range out {
		if u == uriB {
			hasB = true
			break
		}
	}
	if !hasB {
		t.Fatalf("expected peer.ft URI in samePackageOpenURIs, got %v", out)
	}
}

func TestReadForstFilePrefix_truncatesLargeFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "big.ft")
	var b strings.Builder
	b.WriteString("package main\n")
	for i := 0; i < 70000; i++ {
		b.WriteByte('x')
	}
	if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	head, err := readForstFilePrefix(p, 64*1024)
	if err != nil {
		t.Fatal(err)
	}
	if len(head) != 64*1024 {
		t.Fatalf("want 64KiB prefix, got %d", len(head))
	}
}
