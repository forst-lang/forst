package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestLabelDefinitionReferencesHoverCompletion(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)

	dir := t.TempDir()
	ftPath := filepath.Join(dir, "labels.ft")
	const src = `package main

func main() {
	goto done
	println(1)
done:
	println(2)
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	// goto done — definition should jump to done:
	gotoPos := LSPPosition{Line: 3, Character: 6} // "done" in goto done
	loc := s.findDefinitionForPosition(uri, gotoPos)
	if loc == nil {
		t.Fatal("expected definition for goto label")
	}
	if loc.Range.Start.Line != 5 { // done: is on line 6 in 1-based = index 5
		t.Fatalf("definition line = %d, want 5 (done:)", loc.Range.Start.Line)
	}

	refs := s.findReferencesForPosition(uri, gotoPos, true)
	if len(refs) < 2 {
		t.Fatalf("expected decl+use references, got %d", len(refs))
	}

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.TC == nil {
		t.Fatal("analyze failed")
	}
	tok := tokenAtLSPPosition(ctx.Tokens, gotoPos)
	md := labelHoverMarkdown(ctx.TC, ctx.Tokens, tok)
	if md == "" || !strings.Contains(md, "done") {
		t.Fatalf("hover = %q", md)
	}

	items, _ := s.getCompletionsForPosition(uri, LSPPosition{Line: 3, Character: 6}, nil)
	found := false
	for _, it := range items {
		if it.Label == "done" && it.Detail == "label" {
			found = true
			break
		}
	}
	if !found {
		// Completion after "goto " with prefix "done" — character 6 is mid-identifier.
		// Also try right after "goto ".
		items2, _ := s.getCompletionsForPosition(uri, LSPPosition{Line: 3, Character: 5}, nil)
		for _, it := range items2 {
			if it.Label == "done" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected label completion for done, items=%v", items)
	}

	// Diagnostic for undefined goto should not be at 1:1
	bad := `package main
func main() {
	goto missing
}
`
	badPath := filepath.Join(dir, "bad.ft")
	if err := os.WriteFile(badPath, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	badURI := mustFileURI(t, badPath)
	s.documentMu.Lock()
	s.openDocuments[badURI] = bad
	s.documentMu.Unlock()
	diags := s.processForstFile(badURI, bad)
	if len(diags) == 0 {
		t.Fatal("expected diagnostic for undefined label")
	}
	if diags[0].Range.Start.Line == 0 && diags[0].Range.Start.Character == 0 {
		t.Fatalf("diagnostic at 1:1: %+v", diags[0])
	}
}
