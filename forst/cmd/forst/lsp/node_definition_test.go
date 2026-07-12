package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/nodeinterop"

	"github.com/sirupsen/logrus"
)

func writeNodeInteropLSPFixture(t *testing.T) (ftPath, tsPath, src string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module node_def_test\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"node":{"enabled":true,"importPolicy":"explicit"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	legacyDir := filepath.Join(dir, "legacy")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tsPath = filepath.Join(legacyDir, "payment.ts")
	const tsSrc = `export function create(amount: number, currency: string) {
  return { id: "pay_1", amount, currency };
}
`
	if err := os.WriteFile(tsPath, []byte(tsSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	src = `package main

import node payment "./legacy/payment"

func main() {
  payment.create(1.0, "USD")
}
`
	ftPath = filepath.Join(dir, "main.ft")
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return ftPath, tsPath, src
}

func skipUnlessNodeIndexer(t *testing.T, boundaryRoot string) {
	t.Helper()
	if _, err := nodeinterop.RunIndexer(boundaryRoot, []string{"legacy/payment.ts"}); err != nil {
		t.Skipf("node indexer unavailable: %v", err)
	}
}

func TestFindDefinition_nodeImportPathString(t *testing.T) {
	t.Parallel()
	ftPath, tsPath, src := writeNodeInteropLSPFixture(t)
	skipUnlessNodeIndexer(t, filepath.Dir(ftPath))

	log := logrus.New()
	s := NewLSPServer("8080", log)
	uri := mustFileURI(t, ftPath)
	wantTSURI := mustFileURI(t, tsPath)

	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	pos := lspPositionOfStringLiteral(src, `"./legacy/payment"`)
	loc := s.findDefinitionForPosition(uri, pos)
	if loc == nil {
		t.Fatal("expected definition for node import path string")
	}
	if loc.URI != wantTSURI {
		t.Fatalf("definition URI: got %q want %q", loc.URI, wantTSURI)
	}
	if loc.Range.Start.Line != 0 || loc.Range.Start.Character != 0 {
		t.Fatalf("expected file start, got %+v", loc.Range.Start)
	}
}

func TestFindDefinition_nodeImportAlias(t *testing.T) {
	t.Parallel()
	ftPath, tsPath, src := writeNodeInteropLSPFixture(t)
	skipUnlessNodeIndexer(t, filepath.Dir(ftPath))

	s := NewLSPServer("8080", logrus.New())
	uri := mustFileURI(t, ftPath)
	wantTSURI := mustFileURI(t, tsPath)

	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	pos := lspPositionOfIdentifier(src, "payment")
	// first "payment" is in import clause
	lines := strings.Split(src, "\n")
	if !strings.Contains(lines[pos.Line], "import node") {
		t.Fatalf("expected import-clause payment at %+v", pos)
	}

	loc := s.findDefinitionForPosition(uri, pos)
	if loc == nil {
		t.Fatal("expected definition for node import alias")
	}
	if loc.URI != wantTSURI {
		t.Fatalf("definition URI: got %q want %q", loc.URI, wantTSURI)
	}
}

func TestFindDefinition_nodeImportExport(t *testing.T) {
	t.Parallel()
	ftPath, tsPath, src := writeNodeInteropLSPFixture(t)
	skipUnlessNodeIndexer(t, filepath.Dir(ftPath))

	s := NewLSPServer("8080", logrus.New())
	uri := mustFileURI(t, ftPath)
	wantTSURI := mustFileURI(t, tsPath)

	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	pos := lspPositionOfIdentifier(src, "create")
	loc := s.findDefinitionForPosition(uri, pos)
	if loc == nil {
		t.Fatal("expected definition for node export create")
	}
	if loc.URI != wantTSURI {
		t.Fatalf("definition URI: got %q want %q", loc.URI, wantTSURI)
	}
	if loc.Range.Start.Line != 0 {
		t.Fatalf("expected create on line 1 (0-based 0), got line %d", loc.Range.Start.Line)
	}
	if loc.Range.Start.Character != 17 {
		t.Fatalf("expected create at column 17, got %d", loc.Range.Start.Character)
	}
}

func lspPositionOfStringLiteral(content, literal string) LSPPosition {
	lines := strings.Split(content, "\n")
	for li, line := range lines {
		idx := strings.Index(line, literal)
		if idx >= 0 {
			return LSPPosition{Line: li, Character: idx + 1}
		}
	}
	return LSPPosition{}
}

func TestLspLocationPtrFromAbsPath_withDefinitionSpan(t *testing.T) {
	t.Parallel()
	loc := lspLocationPtrFromAbsPath("/tmp/payment.ts", nodeinterop.IndexSourceLocation{
		Line: 1, Column: 17, EndLine: 1, EndColumn: 23,
	})
	if loc == nil {
		t.Fatal("nil location")
	}
	if loc.Range.Start.Line != 0 || loc.Range.Start.Character != 17 {
		t.Fatalf("start = %+v", loc.Range.Start)
	}
	if loc.Range.End.Character != 23 {
		t.Fatalf("end = %+v", loc.Range.End)
	}
}
