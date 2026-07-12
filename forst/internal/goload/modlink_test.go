package goload

import (
	"os"
	"path/filepath"
	"testing"
)

func TestForstModuleLinkFromGoMod_replace(t *testing.T) {
	root := t.TempDir()
	forstGomod := filepath.Join(root, ".forst-gomod")
	if err := os.MkdirAll(forstGomod, 0o755); err != nil {
		t.Fatal(err)
	}
	forstTarget := filepath.Join(root, "vendor", "forst")
	if err := os.MkdirAll(forstTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "module example.com/app/forst\n\ngo 1.26.0\n\nreplace forst => ./vendor/forst\n"
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	link, ok := ForstModuleLinkFromGoMod(forstGomod)
	if !ok {
		t.Fatal("expected link from go.mod")
	}
	want := filepath.Clean(forstTarget)
	if link.ReplaceDir != want {
		t.Fatalf("ReplaceDir = %q want %q", link.ReplaceDir, want)
	}
	if link.RequireVersion != "" {
		t.Fatalf("unexpected RequireVersion %q", link.RequireVersion)
	}
}

func TestForstModuleLinkFromGoMod_require(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module app\n\nrequire forst v0.0.37\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link, ok := ForstModuleLinkFromGoMod(root)
	if !ok {
		t.Fatal("expected link")
	}
	if link.RequireVersion != "v0.0.37" {
		t.Fatalf("RequireVersion = %q want v0.0.37", link.RequireVersion)
	}
	if link.ReplaceDir != "" {
		t.Fatalf("unexpected ReplaceDir %q", link.ReplaceDir)
	}
}

func TestForstModuleLinkFromGoMod_forstGomodUsesBoundaryRoot(t *testing.T) {
	root := t.TempDir()
	forstGomod := filepath.Join(root, ".forst-gomod")
	if err := os.MkdirAll(forstGomod, 0o755); err != nil {
		t.Fatal(err)
	}
	forstTarget := filepath.Join(root, "..", "..", "forst", "forst")
	if err := os.MkdirAll(forstTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "module example.com/app/forst\n\nreplace forst => ../../forst/forst\n"
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	link, ok := ForstModuleLinkFromGoMod(forstGomod)
	if !ok {
		t.Fatal("expected link from go.mod")
	}
	want, err := filepath.Abs(filepath.Join(root, "../../forst/forst"))
	if err != nil {
		t.Fatal(err)
	}
	if link.ReplaceDir != filepath.Clean(want) {
		t.Fatalf("ReplaceDir = %q want %q", link.ReplaceDir, want)
	}
}

func TestForstModuleLinkFromGoMod_replaceOverridesRequire(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "forst-runtime")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "module app\n\nrequire forst v0.0.37\nreplace forst => ./forst-runtime\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	link, ok := ForstModuleLinkFromGoMod(root)
	if !ok {
		t.Fatal("expected link")
	}
	if link.ReplaceDir != filepath.Clean(target) {
		t.Fatalf("ReplaceDir = %q want %q", link.ReplaceDir, target)
	}
}
