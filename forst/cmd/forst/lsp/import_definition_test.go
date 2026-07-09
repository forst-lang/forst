package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"

	"github.com/sirupsen/logrus"
)

func TestForstFileURIsUnderModule_listsDiskFt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module urimod\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ftPath := filepath.Join(dir, "lib.ft")
	if err := os.WriteFile(ftPath, []byte("package lib\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewLSPServer("8080", logrus.New())
	out := s.forstFileURIsUnderModule(dir)
	wantURI := fileURIForLocalPath(ftPath)
	var found bool
	for _, u := range out {
		if u == wantURI {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("forstFileURIsUnderModule=%v want URI %s", out, wantURI)
	}
}

func writeXpkgCrossPackageFixture(t *testing.T, dir string) (authLogPath, apiHandlePath string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("testmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	authDir := filepath.Join(dir, "auth")
	if err := os.MkdirAll(authDir, 0o755); err != nil {
		t.Fatal(err)
	}
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	authLogPath = filepath.Join(authDir, "log.ft")
	apiHandlePath = filepath.Join(apiDir, "handle.ft")
	const srcAuth = `package auth

type Logger = { info(msg String) }

func LogEvent(id String) {
	use logger: Logger
	logger.info("expire " + id)
}
`
	const srcApi = `package api

import "testmod/auth"

func HandleRequest(id String) {
	auth.LogEvent(id)
}
`
	if err := os.WriteFile(authLogPath, []byte(srcAuth), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "stub.go"), []byte("package auth\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(apiHandlePath, []byte(srcApi), 0o644); err != nil {
		t.Fatal(err)
	}
	return authLogPath, apiHandlePath
}

func TestFindDefinition_crossPackageImport_diskPeer(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	authLogPath, apiHandlePath := writeXpkgCrossPackageFixture(t, dir)

	uriAuth := mustFileURI(t, authLogPath)
	uriApi := mustFileURI(t, apiHandlePath)
	srcApi, err := os.ReadFile(apiHandlePath)
	if err != nil {
		t.Fatal(err)
	}

	s.documentMu.Lock()
	s.openDocuments[uriApi] = string(srcApi)
	s.documentMu.Unlock()

	pos := lspPositionOfIdentifier(string(srcApi), "LogEvent")
	loc := s.findDefinitionForPosition(uriApi, pos)
	if loc == nil {
		t.Fatal("expected definition for cross-package LogEvent")
	}
	if loc.URI != uriAuth {
		t.Fatalf("definition URI: got %q want %q", loc.URI, uriAuth)
	}
	if loc.Range.Start.Line != 4 {
		t.Fatalf("definition line: got %d want 4 (func LogEvent)", loc.Range.Start.Line)
	}
}

func TestFindDefinition_crossPackageImport_bothBuffersOpen(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	authLogPath, apiHandlePath := writeXpkgCrossPackageFixture(t, dir)

	uriAuth := mustFileURI(t, authLogPath)
	uriApi := mustFileURI(t, apiHandlePath)
	srcAuth, err := os.ReadFile(authLogPath)
	if err != nil {
		t.Fatal(err)
	}
	srcApi, err := os.ReadFile(apiHandlePath)
	if err != nil {
		t.Fatal(err)
	}

	s.documentMu.Lock()
	s.openDocuments[uriAuth] = string(srcAuth)
	s.openDocuments[uriApi] = string(srcApi)
	s.documentMu.Unlock()

	pos := lspPositionOfIdentifier(string(srcApi), "LogEvent")
	loc := s.findDefinitionForPosition(uriApi, pos)
	if loc == nil {
		t.Fatal("expected definition for cross-package LogEvent")
	}
	if loc.URI != uriAuth {
		t.Fatalf("definition URI: got %q want %q", loc.URI, uriAuth)
	}
}

func TestFindReferences_crossPackageImport_fromDefinition(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	authLogPath, apiHandlePath := writeXpkgCrossPackageFixture(t, dir)

	uriAuth := mustFileURI(t, authLogPath)
	uriApi := mustFileURI(t, apiHandlePath)
	srcAuth, err := os.ReadFile(authLogPath)
	if err != nil {
		t.Fatal(err)
	}
	srcApi, err := os.ReadFile(apiHandlePath)
	if err != nil {
		t.Fatal(err)
	}

	s.documentMu.Lock()
	s.openDocuments[uriAuth] = string(srcAuth)
	s.openDocuments[uriApi] = string(srcApi)
	s.documentMu.Unlock()

	pos := lspPositionOfIdentifier(string(srcAuth), "LogEvent")
	locs := s.findReferencesForPosition(uriAuth, pos, true)
	if len(locs) < 2 {
		t.Fatalf("expected definition + cross-package call reference, got %d: %#v", len(locs), locs)
	}
	seenDef, seenCall := false, false
	for _, l := range locs {
		if l.URI == uriAuth {
			seenDef = true
		}
		if l.URI == uriApi {
			seenCall = true
		}
	}
	if !seenDef {
		t.Fatal("expected LogEvent definition in auth")
	}
	if !seenCall {
		t.Fatal("expected LogEvent call reference in api handle.ft")
	}
}
