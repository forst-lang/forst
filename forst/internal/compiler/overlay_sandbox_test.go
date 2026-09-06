package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/gowork"
)

func TestCreateTempOutputFilesWithOverlays_noCompilerCreatesGoMod(t *testing.T) {
	t.Parallel()
	boundary := t.TempDir()
	overlayDir := filepath.Join(boundary, ".forst", "overlay", "github.com%2Facme%2Flib@local")
	if err := os.MkdirAll(overlayDir, 0o755); err != nil {
		t.Fatal(err)
	}
	replaces := []gowork.PackageReplace{{
		ImportPath: "github.com/acme/lib",
		Dir:        overlayDir,
	}}
	// No bridge/invoke companions => needsCompiler false; fresh sandbox has no go.mod yet.
	out, err := CreateTempOutputFilesWithOverlays(
		"package main\nfunc main() {}\n",
		"", "", nil, nil, boundary, replaces,
	)
	if err != nil {
		t.Fatalf("CreateTempOutputFilesWithOverlays: %v", err)
	}
	goModPath := filepath.Join(filepath.Dir(out), "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("expected sandbox go.mod: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "module forst.run.temp") {
		t.Fatalf("missing module line:\n%s", s)
	}
	if !strings.Contains(s, "replace github.com/acme/lib =>") {
		t.Fatalf("missing overlay replace:\n%s", s)
	}
}
