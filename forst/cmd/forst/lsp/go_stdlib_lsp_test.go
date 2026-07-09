package lsp

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestAnalyzeForstDocument_goStdlib_osGetwd_noUnknownIdentifier(t *testing.T) {
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
	path, uri := importTestModuleFile(t, "os_getwd.ft", src)
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
	diags := s.compileForstFile(path, src, nil)
	for _, d := range diags {
		if strings.Contains(d.Message, "unknown identifier") && strings.Contains(d.Message, "os.Getwd") {
			t.Fatalf("unexpected diagnostic: %s", d.Message)
		}
	}
}

func TestAnalyzeForstDocument_goStdlib_execCommand_noUnknownIdentifier(t *testing.T) {
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
	path, uri := importTestModuleFile(t, "exec_command.ft", src)
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
	diags := s.compileForstFile(path, src, nil)
	assertNoGoImportDiagnostics(t, diags)
	for _, d := range diags {
		if strings.Contains(d.Message, "unknown identifier") && strings.Contains(d.Message, "exec.Command") {
			t.Fatalf("unexpected diagnostic: %s", d.Message)
		}
	}
}

func TestAnalyzeForstDocument_goStdlib_execCommandSliceSpread_noDiagnostics(t *testing.T) {
	src := `package main

import "os/exec"

func main() {
	argv := []String{"true", "extra"}
	exec.Command(argv[0], argv[1:]...)
}
`
	path, uri := importTestModuleFile(t, "exec_spread.ft", src)
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
	diags := s.compileForstFile(path, src, nil)
	for _, d := range diags {
		if strings.Contains(d.Message, "unknown identifier") {
			t.Fatalf("unexpected diagnostic: %s", d.Message)
		}
	}
}

func TestAnalyzeForstDocument_goStdlib_processStateExitCode_noUnknownIdentifier(t *testing.T) {
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
	path, uri := importTestModuleFile(t, "exec_exitcode.ft", src)
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
	diags := s.compileForstFile(path, src, nil)
	for _, d := range diags {
		if strings.Contains(d.Message, "unknown identifier") &&
			(strings.Contains(d.Message, "ProcessState") || strings.Contains(d.Message, "ExitCode")) {
			t.Fatalf("unexpected diagnostic: %s", d.Message)
		}
	}
}

func TestAnalyzeForstDocument_goImportWithoutWorkspace_reportsDiagnostic(t *testing.T) {
	// LSP always sets GoWorkspaceDir from FindModuleRoot (even when no local go.mod).
	// Stdlib packages like os/exec still resolve via go/packages when Dir is set.
	// Verify compileForstFile surfaces structured go-call diagnostics instead.
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
