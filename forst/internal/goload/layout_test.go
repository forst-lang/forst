package goload

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveProjectLayout_flatGoModAtBoundary(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module flat\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	layout, err := ResolveProjectLayout(dir)
	if err != nil {
		t.Fatal(err)
	}
	if layout.Boundary != dir {
		t.Fatalf("Boundary = %q want %q", layout.Boundary, dir)
	}
	if layout.ScanRoot != dir {
		t.Fatalf("ScanRoot = %q want %q", layout.ScanRoot, dir)
	}
	if layout.GoModRoot != dir {
		t.Fatalf("GoModRoot = %q want %q", layout.GoModRoot, dir)
	}
	if layout.ReplaceBase != dir {
		t.Fatalf("ReplaceBase = %q want %q", layout.ReplaceBase, dir)
	}
	if layout.ImportPathRoot != dir {
		t.Fatalf("ImportPathRoot = %q want %q", layout.ImportPathRoot, dir)
	}
}

func TestResolveProjectLayout_forstGomodSubdirLayout(t *testing.T) {
	dir := t.TempDir()
	forstGomod := filepath.Join(dir, ".forst-gomod")
	if err := os.MkdirAll(forstGomod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte("module example.com/app/forst\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	layout, err := ResolveProjectLayout(dir)
	if err != nil {
		t.Fatal(err)
	}
	if layout.ScanRoot != dir {
		t.Fatalf("ScanRoot = %q want boundary %q", layout.ScanRoot, dir)
	}
	if layout.GoModRoot != forstGomod {
		t.Fatalf("GoModRoot = %q want %q", layout.GoModRoot, forstGomod)
	}
	if layout.ReplaceBase != dir {
		t.Fatalf("ReplaceBase = %q want boundary %q", layout.ReplaceBase, dir)
	}
	if layout.ImportPathRoot != dir {
		t.Fatalf("ImportPathRoot = %q want boundary %q", layout.ImportPathRoot, dir)
	}
}

func TestScanRootForPackageRoot_nestedEntryUsesModuleRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module cross_stub\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	got := ScanRootForPackageRoot(apiDir)
	if got != dir {
		t.Fatalf("ScanRootForPackageRoot(%q) = %q, want %q", apiDir, got, dir)
	}
}

func TestGoModReplaceBase_forstGomodUsesBoundary(t *testing.T) {
	dir := t.TempDir()
	forstGomod := filepath.Join(dir, ".forst-gomod")
	got := GoModReplaceBase(forstGomod)
	if got != dir {
		t.Fatalf("GoModReplaceBase = %q want %q", got, dir)
	}
}
