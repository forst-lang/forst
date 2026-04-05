package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func lspPositionOfIdentifier(content, name string) LSPPosition {
	lines := strings.Split(content, "\n")
	for li, line := range lines {
		idx := strings.Index(line, name)
		if idx < 0 {
			continue
		}
		// ensure word boundary: not substring of longer id
		beforeOK := idx == 0 || !isIdentRune(rune(line[idx-1]))
		after := idx + len(name)
		afterOK := after >= len(line) || !isIdentRune(rune(line[after]))
		if beforeOK && afterOK {
			return LSPPosition{Line: li, Character: idx}
		}
	}
	return LSPPosition{}
}

func isIdentRune(r rune) bool {
	return r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}

func TestFindDefinition_crossFileSamePackage(t *testing.T) {
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
  return foo()
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

	pos := lspPositionOfIdentifier(srcB, "foo")
	loc := s.findDefinitionForPosition(uriB, pos)
	if loc == nil {
		t.Fatal("expected definition location for foo")
	}
	if loc.URI != uriA {
		t.Fatalf("definition URI: got %q want %q", loc.URI, uriA)
	}
}

func TestFindReferences_crossFileSamePackage(t *testing.T) {
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
  return foo()
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

	pos := lspPositionOfIdentifier(srcB, "foo")
	locs := s.findReferencesForPosition(uriB, pos, true)
	if len(locs) < 2 {
		t.Fatalf("expected references in both files, got %d", len(locs))
	}
	seenA, seenB := false, false
	for _, l := range locs {
		if l.URI == uriA {
			seenA = true
		}
		if l.URI == uriB {
			seenB = true
		}
	}
	if !seenA || !seenB {
		t.Fatalf("expected refs in a and b, seenA=%v seenB=%v", seenA, seenB)
	}
}

func TestParsePackageGroupMembersParallel_threeFiles(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	var uris []string
	for i := range 3 {
		p := filepath.Join(dir, filepath.FromSlash(string(rune('a'+i))+".ft"))
		c := "package main\n\nfunc f" + string(rune('0'+i)) + "(): Int { return 0 }\n"
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
		u := mustFileURI(t, p)
		uris = append(uris, u)
		s.documentMu.Lock()
		s.openDocuments[u] = c
		s.documentMu.Unlock()
	}

	contentsMap, err := s.loadPackageGroupContents(uris)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.parsePackageGroupMembersParallel(uris, contentsMap)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 3 {
		t.Fatalf("len %d", len(res))
	}
	for i := range res {
		if res[i].ParseErr != nil {
			t.Fatalf("parse err file %d: %v", i, res[i].ParseErr)
		}
		if len(res[i].Nodes) == 0 {
			t.Fatalf("file %d: no nodes", i)
		}
	}
}

func TestProcessForstFile_mergedPackageNoFalseUndefined(t *testing.T) {
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
  return foo()
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

	d := s.processForstFile(uriB, srcB)
	for _, x := range d {
		if strings.Contains(strings.ToLower(x.Message), "undefined") {
			t.Fatalf("unexpected undefined diagnostic: %#v", x)
		}
	}
}

func TestGetCompletions_mergedPackageNoCrossBufferDuplicate(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.ft")
	bPath := filepath.Join(dir, "b.ft")
	const srcA = `package main

func shared(): Int {
  return 1
}
`
	const srcB = `package main

func bar(): Int {
  return 0
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

	pos := LSPPosition{Line: 3, Character: 2}
	items, _ := s.getCompletionsForPosition(uriB, pos, nil)
	seen := 0
	for _, it := range items {
		if it.Label == "shared" {
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("expected exactly one shared completion, got %d", seen)
	}
}
