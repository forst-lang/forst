package modulecheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"
)

func TestScanModule_rejectsSamePackageInSiblingDirectories(t *testing.T) {
	root := t.TempDir()
	authDir := filepath.Join(root, "auth")
	cryptoDir := filepath.Join(root, "auth", "crypto")
	if err := os.MkdirAll(cryptoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(authDir, "a.ft")
	b := filepath.Join(cryptoDir, "b.ft")
	if err := os.WriteFile(a, []byte("package auth\n\nfunc A(): String { return \"a\" }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("package auth\n\nfunc B(): String { return \"b\" }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed := make(map[string][]ast.Node, 2)
	for _, path := range []string{a, b} {
		nodes, err := forstpkg.ParseForstFile(nil, path)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		parsed[path] = nodes
	}

	_, err := ScanModule(nil, Options{ModuleRoot: root, ParsedFiles: parsed, SkipGoLoad: true, SkipValidate: true})
	if err == nil {
		t.Fatal("expected error when same package spans sibling directories")
	}
	if !strings.Contains(err.Error(), `package "auth" spans 2 directories`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScanModule_rejectsMultiplePackagesInSameDirectory(t *testing.T) {
	root := t.TempDir()
	mainPath := filepath.Join(root, "main.ft")
	authPath := filepath.Join(root, "auth.ft")
	if err := os.WriteFile(mainPath, []byte("package main\n\nfunc Main(): String { return \"m\" }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authPath, []byte("package auth\n\nfunc Auth(): String { return \"a\" }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed := make(map[string][]ast.Node, 2)
	for _, path := range []string{mainPath, authPath} {
		nodes, err := forstpkg.ParseForstFile(nil, path)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		parsed[path] = nodes
	}

	_, err := ScanModule(nil, Options{ModuleRoot: root, ParsedFiles: parsed, SkipGoLoad: true, SkipValidate: true})
	if err == nil {
		t.Fatal("expected error when multiple packages share a directory")
	}
	if !strings.Contains(err.Error(), "contains 2 packages") {
		t.Fatalf("unexpected error: %v", err)
	}
}
