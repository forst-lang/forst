package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

// Regression: Providers contract typedefs use method signatures in shapes (`info(msg String)`).
func TestAnalyzeForstDocument_providersExample_methodContractParses(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "examples", "in", "rfc", "providers", "providers.ft"))
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(root)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(root)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module providers_demo\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(filepath.Join(dir, "go.mod")) })

	uri := mustFileURI(t, root)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = string(src)
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatalf("expected analyze ok, got ok=%v ctx=%v", ok, ctx)
	}
	if ctx.ParseErr != nil {
		t.Fatalf("parse providers.ft: %v", ctx.ParseErr)
	}
	if ctx.TC == nil {
		t.Fatal("expected typechecker after successful parse")
	}
}
