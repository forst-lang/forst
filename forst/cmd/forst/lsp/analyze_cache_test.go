package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestAnalyzeForstDocument_diskFallbackWithoutOpenBuffer(t *testing.T) {
	t.Parallel()
	server := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	filePath := filepath.Join(dir, "disk.ft")
	const source = `package main

func main() {}
`
	if err := os.WriteFile(filePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, filePath)

	context, ok := server.analyzeForstDocument(uri)
	if !ok || context == nil {
		t.Fatalf("expected disk fallback analyze success, got ok=%v ctx=%v", ok, context)
	}
	if context.Content != source {
		t.Fatalf("expected context content from disk, got %q", context.Content)
	}
	if context.TC == nil {
		t.Fatal("expected typechecker for valid disk file")
	}
}

func TestAnalyzeForstDocument_returnsFalseWhenDebuggerUnavailable(t *testing.T) {
	t.Parallel()
	server := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	filePath := filepath.Join(dir, "x.ft")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, filePath)
	server.debugger = nil

	context, ok := server.analyzeForstDocument(uri)
	if ok || context != nil {
		t.Fatalf("expected analyze failure without debugger, got ok=%v ctx=%v", ok, context)
	}
}

func TestPeerDocumentContextForCompletion_cacheReuseAndInvalidation(t *testing.T) {
	t.Parallel()
	server := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	filePath := filepath.Join(dir, "peer.ft")
	const source = `package main

func Peer(): Int {
	return 1
}
`
	if err := os.WriteFile(filePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, filePath)
	server.documentMu.Lock()
	server.openDocuments[uri] = source
	server.documentMu.Unlock()

	firstContext, firstOK := server.peerDocumentContextForCompletion(uri)
	if !firstOK || firstContext == nil {
		t.Fatalf("expected first peer analysis context, got ok=%v ctx=%v", firstOK, firstContext)
	}

	secondContext, secondOK := server.peerDocumentContextForCompletion(uri)
	if !secondOK || secondContext == nil {
		t.Fatalf("expected second peer analysis context, got ok=%v ctx=%v", secondOK, secondContext)
	}
	if firstContext != secondContext {
		t.Fatal("expected peer context pointer reuse from cache for unchanged buffer")
	}

	server.invalidatePeerAnalysisCache(uri)
	thirdContext, thirdOK := server.peerDocumentContextForCompletion(uri)
	if !thirdOK || thirdContext == nil {
		t.Fatalf("expected peer analysis context after invalidation, got ok=%v ctx=%v", thirdOK, thirdContext)
	}
	if thirdContext == secondContext {
		t.Fatal("expected fresh peer context pointer after cache invalidation")
	}
}
