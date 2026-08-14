package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolutionParity_snapshot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module paritytest\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"server":{"embedded":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	mainDir := filepath.Join(dir, "main")
	bcryptDir := filepath.Join(dir, "bcrypt")
	for _, d := range []string{mainDir, bcryptDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(mainDir, "main.ft"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bcryptDir, "bcrypt.ft"), []byte(`package bcrypt

func Hash() { return {h: "x"} }
`), 0o644); err != nil {
		t.Fatal(err)
	}

	proj, err := Open(nil, OpenOpts{BoundaryRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	pkgs := proj.ForstPackages()
	if len(pkgs) < 2 {
		t.Fatalf("expected 2+ packages, got %v", pkgs)
	}
	runnable, err := proj.RunnableFunctions()
	if err != nil {
		t.Fatal(err)
	}
	foundBcrypt := false
	for _, fn := range runnable {
		if fn.Package == "bcrypt" && fn.Name == "Hash" {
			foundBcrypt = true
		}
	}
	if !foundBcrypt {
		t.Fatalf("expected bcrypt.Hash in runnable set: %v", runnable)
	}
	mod := proj.Module
	if mod == nil || len(mod.ForstPkgToFiles) < 2 {
		t.Fatal("expected module graph with multiple packages")
	}
	for _, p := range pkgs {
		if !strings.Contains(strings.Join(pkgs, ","), p) {
			t.Fatalf("package %q missing from sorted list", p)
		}
	}
}

func TestResolutionParity_forstGomodSubdirLayout(t *testing.T) {
	dir := t.TempDir()
	forstGomod := filepath.Join(dir, ".forst-gomod")
	if err := os.MkdirAll(forstGomod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte("module example.com/app/forst\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	forstDir := filepath.Join(dir, "forst")
	mainDir := filepath.Join(forstDir, "main")
	helperDir := filepath.Join(forstDir, "helper")
	for _, d := range []string{mainDir, helperDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(mainDir, "main.ft"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helperDir, "helper.ft"), []byte(`package helper

func Ping() {
  return {ok: "yes"}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	proj, err := Open(nil, OpenOpts{BoundaryRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	if proj.ModuleRoot != forstGomod {
		t.Fatalf("ModuleRoot = %q want %q", proj.ModuleRoot, forstGomod)
	}
	runnable, err := proj.RunnableFunctions()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, fn := range runnable {
		if fn.Package == "helper" && fn.Name == "Ping" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected helper.Ping in runnable set: %v", runnable)
	}
}
