package lsp

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestAnalyzeForstDocument_goStdlib_osGetwd_noUnknownIdentifier(t *testing.T) {
	t.Parallel()
	src := `package main

import "os"

func cwdProbe(): String {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}
`
	_, uri := importTestModuleFile(t, "os_getwd.ft", src)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("analyze failed")
	}
	if ctx.CheckErr != nil {
		t.Fatalf("unexpected typecheck error: %v", ctx.CheckErr)
	}
}

func TestAnalyzeForstDocument_goStdlib_execCommand_noUnknownIdentifier(t *testing.T) {
	t.Parallel()
	src := `package main

import "os/exec"

func execProbe(): Int {
	cmd := exec.Command("true")
	err := cmd.Run()
	if err != nil {
		return 1
	}
	return 0
}
`
	_, uri := importTestModuleFile(t, "exec_command.ft", src)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("analyze failed")
	}
	if ctx.CheckErr != nil {
		t.Fatalf("unexpected typecheck error: %v", ctx.CheckErr)
	}
}

func TestAnalyzeForstDocument_goStdlib_execCommandSliceSpread_noDiagnostics(t *testing.T) {
	t.Parallel()
	src := `package main

import "os/exec"

func main() {
	argv := []String{"true", "extra"}
	exec.Command(argv[0], argv[1:]...)
}
`
	_, uri := importTestModuleFile(t, "exec_spread.ft", src)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("analyze failed")
	}
	if ctx.CheckErr != nil {
		t.Fatalf("unexpected typecheck error: %v", ctx.CheckErr)
	}
	if ctx.TC != nil && !ctx.TC.GoImportPackageLoaded("exec") {
		t.Skip("os/exec not loaded")
	}
}

func TestAnalyzeForstDocument_goStdlib_processStateExitCode_noUnknownIdentifier(t *testing.T) {
	t.Parallel()
	src := `package main

import "os/exec"

func exitCode(argv []String): Int {
	cmd := exec.Command(argv[0], argv[1:]...)
	err := cmd.Run()
	if err != nil {
		if cmd.ProcessState != nil {
			return cmd.ProcessState.ExitCode()
		}
		return 2
	}
	return 0
}
`
	_, uri := importTestModuleFile(t, "exec_exitcode.ft", src)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("analyze failed")
	}
	if ctx.CheckErr != nil {
		t.Fatalf("unexpected typecheck error: %v", ctx.CheckErr)
	}
	if ctx.TC != nil && !ctx.TC.GoImportPackageLoaded("exec") {
		t.Skip("os/exec not loaded")
	}
}

func TestCompileForstFile_goImportBadSpread_reportsGoCallDiagnostic(t *testing.T) {
	t.Parallel()
	// Dedicated compile-path test: verify compileForstFile surfaces structured go-call diagnostics.
	src := `package main

import "os/exec"

func main() {
	exec.Command("true", [1, 2]...)
}
`
	path, uri := importTestModuleFile(t, "exec_bad_spread.ft", src)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("analyze failed")
	}
	if ctx.TC != nil && !ctx.TC.GoImportPackageLoaded("exec") {
		t.Skip("os/exec not loaded")
	}
	if ctx.CheckErr == nil {
		t.Fatal("expected typecheck error for bad spread element type")
	}
	diags := s.compileForstFile(path, src, nil)
	found := false
	for _, d := range diags {
		if d.Code == "go-call" || strings.Contains(d.Message, "cannot spread") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected go-call diagnostic in compile output, checkErr=%v diags=%v", ctx.CheckErr, diags)
	}
}
