package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/goload"
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
	const srcAPI = `package api

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
	if err := os.WriteFile(apiHandlePath, []byte(srcAPI), 0o644); err != nil {
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
	uriAPI := mustFileURI(t, apiHandlePath)
	srcAPI, err := os.ReadFile(apiHandlePath)
	if err != nil {
		t.Fatal(err)
	}

	s.documentMu.Lock()
	s.openDocuments[uriAPI] = string(srcAPI)
	s.documentMu.Unlock()

	pos := lspPositionOfIdentifier(string(srcAPI), "LogEvent")
	loc := s.findDefinitionForPosition(uriAPI, pos)
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
	uriAPI := mustFileURI(t, apiHandlePath)
	srcAuth, err := os.ReadFile(authLogPath)
	if err != nil {
		t.Fatal(err)
	}
	srcAPI, err := os.ReadFile(apiHandlePath)
	if err != nil {
		t.Fatal(err)
	}

	s.documentMu.Lock()
	s.openDocuments[uriAuth] = string(srcAuth)
	s.openDocuments[uriAPI] = string(srcAPI)
	s.documentMu.Unlock()

	pos := lspPositionOfIdentifier(string(srcAPI), "LogEvent")
	loc := s.findDefinitionForPosition(uriAPI, pos)
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
	uriAPI := mustFileURI(t, apiHandlePath)
	srcAuth, err := os.ReadFile(authLogPath)
	if err != nil {
		t.Fatal(err)
	}
	srcAPI, err := os.ReadFile(apiHandlePath)
	if err != nil {
		t.Fatal(err)
	}

	s.documentMu.Lock()
	s.openDocuments[uriAuth] = string(srcAuth)
	s.openDocuments[uriAPI] = string(srcAPI)
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
		if l.URI == uriAPI {
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

func TestFindDefinition_externalForstDep(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)
	root := t.TempDir()
	consumer := filepath.Join(root, "consumer")
	lib := filepath.Join(root, "acme-lib")
	if err := os.MkdirAll(consumer, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	libAdd := filepath.Join(lib, "add.ft")
	mainPath := filepath.Join(consumer, "main.ft")
	if err := os.WriteFile(filepath.Join(lib, "go.mod"), []byte("module github.com/acme/lib\n\ngo "+testmod.GoVersion+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(libAdd, []byte("package lib\n\nfunc Add(a Int, b Int) {\n\treturn a + b\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(consumer, "go.mod"), []byte(testmod.GoModContent("testmod")+"\nrequire github.com/acme/lib v0.0.0\n\nreplace github.com/acme/lib => ../acme-lib\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srcMain := "package main\n\nimport \"github.com/acme/lib\"\n\nfunc main() {\n\t_ = lib.Add(1, 2)\n}\n"
	if err := os.WriteFile(mainPath, []byte(srcMain), 0o644); err != nil {
		t.Fatal(err)
	}

	goload.ClearLoadCacheForTest()
	uriMain := mustFileURI(t, mainPath)
	s.documentMu.Lock()
	s.openDocuments[uriMain] = srcMain
	s.documentMu.Unlock()

	pos := lspPositionOfIdentifier(srcMain, "Add")
	loc := s.findDefinitionForPosition(uriMain, pos)
	if loc == nil {
		t.Fatal("expected definition for external lib.Add")
	}
	gotPath := filePathFromDocumentURI(loc.URI)
	gotEval, err := filepath.EvalSymlinks(gotPath)
	if err != nil {
		gotEval = gotPath
	}
	wantEval, err := filepath.EvalSymlinks(libAdd)
	if err != nil {
		wantEval = libAdd
	}
	if gotEval != wantEval {
		t.Fatalf("definition path: got %q want %q", gotPath, libAdd)
	}
}
