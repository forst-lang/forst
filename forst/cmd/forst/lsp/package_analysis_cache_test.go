package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestAnalyzePackageGroupMerged_parseErrorInPeerDisablesMergedMode(t *testing.T) {
	t.Parallel()
	server := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	goodPath := filepath.Join(dir, "good.ft")
	badPath := filepath.Join(dir, "bad.ft")
	const goodSource = `package main

func Good(): Int {
	return 1
}
`
	const badSource = `package main

unexpected
`
	if err := os.WriteFile(goodPath, []byte(goodSource), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badPath, []byte(badSource), 0o644); err != nil {
		t.Fatal(err)
	}
	goodURI := mustFileURI(t, goodPath)
	badURI := mustFileURI(t, badPath)

	server.documentMu.Lock()
	server.openDocuments[goodURI] = goodSource
	server.openDocuments[badURI] = badSource
	server.documentMu.Unlock()

	snapshot, context, ok := server.analyzePackageGroupMerged(goodURI)
	if ok || snapshot != nil || context != nil {
		t.Fatalf("expected merged analysis to disable on peer parse error, got ok=%v snap=%v ctx=%v", ok, snapshot, context)
	}
}

func TestAnalyzePackageGroupMerged_reusesCachedSnapshotForSameFingerprint(t *testing.T) {
	t.Parallel()
	server := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "a.ft")
	secondPath := filepath.Join(dir, "b.ft")
	const firstSource = `package main

func A(): Int {
	return 1
}
`
	const secondSource = `package main

func B(): Int {
	return 2
}
`
	if err := os.WriteFile(firstPath, []byte(firstSource), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondPath, []byte(secondSource), 0o644); err != nil {
		t.Fatal(err)
	}
	firstURI := mustFileURI(t, firstPath)
	secondURI := mustFileURI(t, secondPath)

	server.documentMu.Lock()
	server.openDocuments[firstURI] = firstSource
	server.openDocuments[secondURI] = secondSource
	server.documentMu.Unlock()

	firstSnapshot, firstContext, firstOK := server.analyzePackageGroupMerged(firstURI)
	if !firstOK || firstSnapshot == nil || firstContext == nil {
		t.Fatalf("expected first merged snapshot, got ok=%v snap=%v ctx=%v", firstOK, firstSnapshot, firstContext)
	}

	secondSnapshot, secondContext, secondOK := server.analyzePackageGroupMerged(firstURI)
	if !secondOK || secondSnapshot == nil || secondContext == nil {
		t.Fatalf("expected second merged snapshot, got ok=%v snap=%v ctx=%v", secondOK, secondSnapshot, secondContext)
	}
	if firstSnapshot != secondSnapshot {
		t.Fatal("expected cached snapshot pointer to be reused for unchanged fingerprint")
	}
}

func TestLoadPackageGroupContents_diskFallbackError(t *testing.T) {
	t.Parallel()
	server := NewLSPServer("8080", logrus.New())
	missingURI := mustFileURI(t, filepath.Join(t.TempDir(), "missing.ft"))

	_, err := server.loadPackageGroupContents([]string{missingURI})
	if err == nil {
		t.Fatal("expected loadPackageGroupContents to fail for missing disk file")
	}
}
