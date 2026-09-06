package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestPublishPeerDiagnosticsFromGroup_doesNotReprocessPeers(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.ft")
	bPath := filepath.Join(dir, "b.ft")
	const srcA = `package main

func A(): Int {
	return 1
}
`
	const srcB = `package main

func B(): Int {
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
	s.openDocuments[uriB] = srcB
	s.documentMu.Unlock()

	uris := s.samePackageOpenURIs(uriA)
	if len(uris) < 2 {
		t.Fatalf("want package group, got %v", uris)
	}

	diags := s.processForstFileWithURIs(uriA, srcA)
	before := s.processForstFileInvocations.Load()
	s.publishPeerDiagnosticsFromGroup(uris, uriA, diags)
	after := s.processForstFileInvocations.Load()
	if after != before {
		t.Fatalf("publishPeerDiagnosticsFromGroup reprocessed peers: invocations %d → %d", before, after)
	}
}

func TestParsePackageGroupMembersParallel_reusesUnchangedPeerParse(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.ft")
	bPath := filepath.Join(dir, "b.ft")
	const srcA = `package main

func A(): Int {
	return 1
}
`
	const srcB = `package main

func B(): Int {
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
	uris := []string{uriA, uriB}
	contents := map[string]string{uriA: srcA, uriB: srcB}

	first, err := s.parsePackageGroupMembersParallel(uris, contents)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 {
		t.Fatalf("want 2 results, got %d", len(first))
	}
	firstParses := s.fileParseInvocations.Load()
	if firstParses != 2 {
		t.Fatalf("first pass: want 2 parses, got %d", firstParses)
	}

	editedA := `package main

func A(): Int {
	return 99
}
`
	contents2 := map[string]string{uriA: editedA, uriB: srcB}
	second, err := s.parsePackageGroupMembersParallel(uris, contents2)
	if err != nil {
		t.Fatal(err)
	}
	secondParses := s.fileParseInvocations.Load()
	if secondParses != 3 {
		t.Fatalf("after editing A only: want 3 total parses (reuse B), got %d", secondParses)
	}
	if second[1].Content != srcB {
		t.Fatalf("peer B content = %q, want unchanged", second[1].Content)
	}
	// Cached peer should keep the same token slice identity from the first parse.
	if &second[1].Tokens[0] != &first[1].Tokens[0] {
		t.Fatal("expected unchanged peer to reuse cached token slice")
	}
}

func TestFileParseCache_invalidateOnRemove(t *testing.T) {
	t.Parallel()
	c := newFileParseCache(8)
	uri := "file:///tmp/x.ft"
	hash := contentSHA256("package main\n")
	c.put(uri, hash, fileParseResult{URI: uri, Content: "package main\n"})
	if _, ok := c.get(uri, hash); !ok {
		t.Fatal("expected cache hit")
	}
	c.remove(uri)
	if _, ok := c.get(uri, hash); ok {
		t.Fatal("expected miss after remove")
	}
}
