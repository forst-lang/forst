package bridgeinterop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeModuleID_precompiled(t *testing.T) {
	got := RuntimeModuleID("legacy/payment.ts", ".forst/js")
	want := ".forst/js/legacy/payment.js"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRemapManifestModuleIDs(t *testing.T) {
	m := ManifestV1{
		Version: 1,
		Exports: []ExportEntry{{ModuleID: "legacy/a.ts", Name: "f", Kind: ExportKindFunction}},
	}
	got := RemapManifestModuleIDs(m, ".forst/js")
	if got.Exports[0].ModuleID != ".forst/js/legacy/a.js" {
		t.Fatalf("moduleId: %q", got.Exports[0].ModuleID)
	}
}

func TestCopyJSArtifacts(t *testing.T) {
	dir := t.TempDir()
	srcRoot := filepath.Join(dir, ".forst", "js", "legacy")
	if err := os.MkdirAll(srcRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcRoot, "a.js"), []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	if err := CopyJSArtifacts(dir, dest, ".forst/js"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, ".forst", "js", "legacy", "a.js")); err != nil {
		t.Fatal(err)
	}
}
