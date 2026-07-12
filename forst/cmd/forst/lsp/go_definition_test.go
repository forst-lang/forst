package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

const goInteropFtSource = `package main

import "os/exec"

func runExecDemo() {
	argv := []String{"true", "extra"}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Run()
}

func runCustomDemo() {
	sum := AddInts(40, 2)
	println(sum)
	message := GreetUpper("forst")
	println(message)
}
`

func writeGoInteropLSPFixture(t *testing.T) (ftPath, helpersPath, src string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module go_interop\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const helpersGo = `package main

import "strings"

// GreetUpper is hand-written Go exported for same-package Forst calls.
func GreetUpper(name string) string {
	return strings.ToUpper(name)
}

// AddInts is hand-written Go exported for same-package Forst calls.
func AddInts(a, b int) int {
	return a + b
}
`
	helpersPath = filepath.Join(dir, "helpers.go")
	if err := os.WriteFile(helpersPath, []byte(helpersGo), 0o644); err != nil {
		t.Fatal(err)
	}
	ftPath = filepath.Join(dir, "cli.ft")
	if err := os.WriteFile(ftPath, []byte(goInteropFtSource), 0o644); err != nil {
		t.Fatal(err)
	}
	return ftPath, helpersPath, goInteropFtSource
}

func skipUnlessGoImportLoaded(t *testing.T, s *LSPServer, uri, src, local string) {
	t.Helper()
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.TC == nil || !ctx.TC.GoImportPackageLoaded(local) {
		t.Skipf("Go import %q not loaded", local)
	}
}

func TestFindDefinition_goQualifiedExport_command(t *testing.T) {
	t.Parallel()
	ftPath, _, src := writeGoInteropLSPFixture(t)
	s := NewLSPServer("8080", logrus.New())
	uri := mustFileURI(t, ftPath)
	skipUnlessGoImportLoaded(t, s, uri, src, "exec")

	pos := lspPositionOfIdentifier(src, "Command")
	loc := s.findDefinitionForPosition(uri, pos)
	if loc == nil {
		t.Fatal("expected definition for exec.Command")
	}
	if !strings.Contains(loc.URI, "exec") {
		t.Fatalf("definition URI: %q", loc.URI)
	}
	if loc.Range.Start.Line < 0 {
		t.Fatalf("line: %+v", loc.Range.Start)
	}
}

func TestFindDefinition_goSamePackageFunc_addInts(t *testing.T) {
	t.Parallel()
	ftPath, helpersPath, src := writeGoInteropLSPFixture(t)
	s := NewLSPServer("8080", logrus.New())
	uri := mustFileURI(t, ftPath)
	wantURI := mustFileURI(t, helpersPath)

	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	pos := lspPositionOfIdentifier(src, "AddInts")
	loc := s.findDefinitionForPosition(uri, pos)
	if loc == nil {
		t.Fatal("expected definition for AddInts")
	}
	if loc.URI != wantURI {
		t.Fatalf("definition URI: got %q want %q", loc.URI, wantURI)
	}
	if loc.Range.Start.Line != 8 {
		t.Fatalf("expected AddInts at line 9 (0-based 8), got %d", loc.Range.Start.Line)
	}
}

func TestFindDefinition_goReceiverMethod_run(t *testing.T) {
	t.Parallel()
	const src = `package main

import "os/exec"

func main() {
	cmd := exec.Command("true")
	cmd.Run()
}
`
	ftPath := filepath.Join(t.TempDir(), "main.ft")
	if err := os.WriteFile(filepath.Join(filepath.Dir(ftPath), "go.mod"), []byte("module go_interop\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewLSPServer("8080", logrus.New())
	uri := mustFileURI(t, ftPath)
	skipUnlessGoImportLoaded(t, s, uri, src, "exec")

	if ctx, ok := s.analyzeForstDocument(uri); !ok || ctx.TC == nil || ctx.TC.GoTypeForVariable("cmd") == nil {
		t.Fatal("expected Go type for cmd in LSP analyze")
	}

	pos := lspPositionOfIdentifier(src, "Run")
	loc := s.findDefinitionForPosition(uri, pos)
	if loc == nil {
		t.Fatal("expected definition for cmd.Run")
	}
	if !strings.Contains(loc.URI, "exec") {
		t.Fatalf("definition URI: %q", loc.URI)
	}
}

func TestFindDefinition_goImportPathString(t *testing.T) {
	t.Parallel()
	ftPath, _, src := writeGoInteropLSPFixture(t)
	s := NewLSPServer("8080", logrus.New())
	uri := mustFileURI(t, ftPath)
	skipUnlessGoImportLoaded(t, s, uri, src, "exec")

	pos := lspPositionOfStringLiteral(src, `"os/exec"`)
	loc := s.findDefinitionForPosition(uri, pos)
	if loc == nil {
		t.Fatal("expected definition for import path string")
	}
	if !strings.Contains(loc.URI, "exec") {
		t.Fatalf("definition URI: %q", loc.URI)
	}
}
