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
	if err := os.WriteFile(filepath.Join(dir, "main.ft"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bcrypt.ft"), []byte(`package bcrypt

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
