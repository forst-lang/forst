package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func writeXpkgCrossPackageFixture(t *testing.T, dir string) (alphaLogPath, betaHandlePath string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	alphaDir := filepath.Join(dir, "alpha")
	if err := os.MkdirAll(alphaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	betaDir := filepath.Join(dir, "beta")
	if err := os.MkdirAll(betaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alphaLogPath = filepath.Join(alphaDir, "log.ft")
	betaHandlePath = filepath.Join(betaDir, "handle.ft")
	const srcAlpha = `package alpha

type Logger = { info(msg String) }

func LogExpiry(id String) {
	use logger: Logger
	logger.info("expire " + id)
}
`
	const srcBeta = `package beta

import "testmod/alpha"

func Handle(id String) {
	alpha.LogExpiry(id)
}
`
	if err := os.WriteFile(alphaLogPath, []byte(srcAlpha), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(alphaDir, "stub.go"), []byte("package alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(betaHandlePath, []byte(srcBeta), 0o644); err != nil {
		t.Fatal(err)
	}
	return alphaLogPath, betaHandlePath
}

func TestFindDefinition_crossPackageImport_diskPeer(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	alphaLogPath, betaHandlePath := writeXpkgCrossPackageFixture(t, dir)

	uriAlpha := mustFileURI(t, alphaLogPath)
	uriBeta := mustFileURI(t, betaHandlePath)
	srcBeta, err := os.ReadFile(betaHandlePath)
	if err != nil {
		t.Fatal(err)
	}

	s.documentMu.Lock()
	s.openDocuments[uriBeta] = string(srcBeta)
	s.documentMu.Unlock()

	pos := lspPositionOfIdentifier(string(srcBeta), "LogExpiry")
	loc := s.findDefinitionForPosition(uriBeta, pos)
	if loc == nil {
		t.Fatal("expected definition for cross-package LogExpiry")
	}
	if loc.URI != uriAlpha {
		t.Fatalf("definition URI: got %q want %q", loc.URI, uriAlpha)
	}
	if loc.Range.Start.Line != 4 {
		t.Fatalf("definition line: got %d want 4 (func LogExpiry)", loc.Range.Start.Line)
	}
}

func TestFindDefinition_crossPackageImport_bothBuffersOpen(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	alphaLogPath, betaHandlePath := writeXpkgCrossPackageFixture(t, dir)

	uriAlpha := mustFileURI(t, alphaLogPath)
	uriBeta := mustFileURI(t, betaHandlePath)
	srcAlpha, err := os.ReadFile(alphaLogPath)
	if err != nil {
		t.Fatal(err)
	}
	srcBeta, err := os.ReadFile(betaHandlePath)
	if err != nil {
		t.Fatal(err)
	}

	s.documentMu.Lock()
	s.openDocuments[uriAlpha] = string(srcAlpha)
	s.openDocuments[uriBeta] = string(srcBeta)
	s.documentMu.Unlock()

	pos := lspPositionOfIdentifier(string(srcBeta), "LogExpiry")
	loc := s.findDefinitionForPosition(uriBeta, pos)
	if loc == nil {
		t.Fatal("expected definition for cross-package LogExpiry")
	}
	if loc.URI != uriAlpha {
		t.Fatalf("definition URI: got %q want %q", loc.URI, uriAlpha)
	}
}

func TestFindReferences_crossPackageImport_fromDefinition(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	alphaLogPath, betaHandlePath := writeXpkgCrossPackageFixture(t, dir)

	uriAlpha := mustFileURI(t, alphaLogPath)
	uriBeta := mustFileURI(t, betaHandlePath)
	srcAlpha, err := os.ReadFile(alphaLogPath)
	if err != nil {
		t.Fatal(err)
	}
	srcBeta, err := os.ReadFile(betaHandlePath)
	if err != nil {
		t.Fatal(err)
	}

	s.documentMu.Lock()
	s.openDocuments[uriAlpha] = string(srcAlpha)
	s.openDocuments[uriBeta] = string(srcBeta)
	s.documentMu.Unlock()

	pos := lspPositionOfIdentifier(string(srcAlpha), "LogExpiry")
	locs := s.findReferencesForPosition(uriAlpha, pos, true)
	if len(locs) < 2 {
		t.Fatalf("expected definition + cross-package call reference, got %d: %#v", len(locs), locs)
	}
	seenDef, seenCall := false, false
	for _, l := range locs {
		if l.URI == uriAlpha {
			seenDef = true
		}
		if l.URI == uriBeta {
			seenCall = true
		}
	}
	if !seenDef {
		t.Fatal("expected LogExpiry definition in alpha")
	}
	if !seenCall {
		t.Fatal("expected LogExpiry call reference in beta handle.ft")
	}
}
