package goload

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindModuleRoot_findsAncestor(t *testing.T) {
	root := t.TempDir()
	modDir := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testmod\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ft := filepath.Join(modDir, "x.ft")
	if err := os.WriteFile(ft, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := FindModuleRoot(ft); got != root {
		t.Fatalf("FindModuleRoot(file): got %q want %q", got, root)
	}
	if got := FindModuleRoot(modDir); got != root {
		t.Fatalf("FindModuleRoot(dir): got %q want %q", got, root)
	}
}

func TestModuleRootWithGoMod_findsAncestor(t *testing.T) {
	root := t.TempDir()
	modDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testmod\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ModuleRootWithGoMod(modDir)
	if err != nil {
		t.Fatalf("ModuleRootWithGoMod: %v", err)
	}
	if got != root {
		t.Fatalf("got %q want %q", got, root)
	}
}

func TestModuleRootWithGoMod_missingGoMod_errors(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ModuleRootWithGoMod(sub); err == nil {
		t.Fatal("expected error when no go.mod exists")
	}
}

func TestFindModuleRoot_noGoMod_fallsBackToStartDir(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := FindModuleRoot(sub); got != sub {
		t.Fatalf("got %q want %q", got, sub)
	}
}

func TestFindModuleRoot_forstGomod(t *testing.T) {
	root := t.TempDir()
	forstGomod := filepath.Join(root, ".forst-gomod")
	if err := os.MkdirAll(forstGomod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte("module example.com/app/forst\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := FindModuleRoot(root); got != forstGomod {
		t.Fatalf("FindModuleRoot(root): got %q want %q", got, forstGomod)
	}
	if got := ModulePath(forstGomod); got != "example.com/app/forst" {
		t.Fatalf("ModulePath: got %q want example.com/app/forst", got)
	}
}

func TestModuleRootHasGoMod_forstGomod(t *testing.T) {
	root := t.TempDir()
	forstGomod := filepath.Join(root, ".forst-gomod")
	if err := os.MkdirAll(forstGomod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte("module example.com/app/forst\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !ModuleRootHasGoMod(root) {
		t.Fatal("expected true when .forst-gomod/go.mod exists")
	}
}

func TestMissingGoModuleSetupHint_noGoMod(t *testing.T) {
	root := t.TempDir()
	hint := MissingGoModuleSetupHint(root)
	if hint == "" {
		t.Fatal("expected hint when no go.mod")
	}
	if !strings.Contains(hint, ".forst-gomod/go.mod") {
		t.Fatalf("hint should mention .forst-gomod/go.mod: %q", hint)
	}
}

func TestMissingGoModuleSetupHint_withForstGomod(t *testing.T) {
	root := t.TempDir()
	forstGomod := filepath.Join(root, ".forst-gomod")
	if err := os.MkdirAll(forstGomod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte("module example.com/app/forst\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if hint := MissingGoModuleSetupHint(root); hint != "" {
		t.Fatalf("expected empty hint, got %q", hint)
	}
}

func writeForstCompilerModuleMarker(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "cmd", "forst"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestIsForstCompilerModule(t *testing.T) {
	root := t.TempDir()
	if IsForstCompilerModule(root) {
		t.Fatal("empty module dir should be false")
	}
	writeForstCompilerModuleMarker(t, root)
	if !IsForstCompilerModule(root) {
		t.Fatal("expected true when cmd/forst exists")
	}
}

func TestForstCompilerModuleRoot_envOverride(t *testing.T) {
	root := t.TempDir()
	writeForstCompilerModuleMarker(t, root)
	t.Setenv(EnvForstGOModRoot, root)
	if got := ForstCompilerModuleRoot(); got != root {
		t.Fatalf("got %q want %q", got, root)
	}
}

func TestForstCompilerModuleRoot_fromBinLayout(t *testing.T) {
	repo := t.TempDir()
	forstMod := filepath.Join(repo, "forst")
	writeForstCompilerModuleMarker(t, forstMod)
	binDir := filepath.Join(repo, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(binDir, "forst")
	if err := os.WriteFile(binary, []byte("fake\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	origExe := modRootExecutable
	origGetwd := modRootGetwd
	t.Cleanup(func() {
		modRootExecutable = origExe
		modRootGetwd = origGetwd
	})
	modRootExecutable = func() (string, error) { return binary, nil }
	modRootGetwd = func() (string, error) { return filepath.Join(repo, "examples", "in", "rfc"), nil }

	if got := ForstCompilerModuleRoot(); evalPath(t, got) != evalPath(t, forstMod) {
		t.Fatalf("got %q want %q", got, forstMod)
	}
}

func evalPath(t *testing.T, p string) string {
	t.Helper()
	if p == "" {
		return ""
	}
	out, err := filepath.EvalSymlinks(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return filepath.Clean(out)
}

func TestForstCompilerModuleRoot_binaryAdjacentModule(t *testing.T) {
	cache := t.TempDir()
	versionDir := filepath.Join(cache, "0.0.37")
	moduleDir := filepath.Join(versionDir, "module")
	writeForstCompilerModuleMarker(t, moduleDir)
	binary := filepath.Join(versionDir, "forst-darwin-arm64")
	if err := os.WriteFile(binary, []byte("fake\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	origExe := modRootExecutable
	origGetwd := modRootGetwd
	t.Cleanup(func() {
		modRootExecutable = origExe
		modRootGetwd = origGetwd
	})
	modRootExecutable = func() (string, error) { return binary, nil }
	modRootGetwd = func() (string, error) { return t.TempDir(), nil }

	if got := ForstCompilerModuleRoot(); evalPath(t, got) != evalPath(t, moduleDir) {
		t.Fatalf("got %q want %q", got, moduleDir)
	}
}

func TestForstCompilerModuleRoot_noMatch(t *testing.T) {
	origExe := modRootExecutable
	origGetwd := modRootGetwd
	t.Cleanup(func() {
		modRootExecutable = origExe
		modRootGetwd = origGetwd
	})
	unrelated := t.TempDir()
	modRootExecutable = func() (string, error) {
		p := filepath.Join(unrelated, "forst")
		if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return p, nil
	}
	modRootGetwd = func() (string, error) { return unrelated, nil }
	t.Setenv(EnvForstGOModRoot, "")

	if got := ForstCompilerModuleRoot(); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestForstCompilerModuleRoot_symlinkedBinary(t *testing.T) {
	repo := t.TempDir()
	forstMod := filepath.Join(repo, "forst")
	writeForstCompilerModuleMarker(t, forstMod)
	binDir := filepath.Join(repo, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	realBin := filepath.Join(binDir, "forst.real")
	if err := os.WriteFile(realBin, []byte("fake\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(binDir, "forst")
	if err := os.Symlink(realBin, linkPath); err != nil {
		t.Fatal(err)
	}

	externDir := t.TempDir()
	externLink := filepath.Join(externDir, "forst")
	if err := os.Symlink(linkPath, externLink); err != nil {
		t.Fatal(err)
	}

	origExe := modRootExecutable
	origGetwd := modRootGetwd
	t.Cleanup(func() {
		modRootExecutable = origExe
		modRootGetwd = origGetwd
	})
	modRootExecutable = func() (string, error) { return externLink, nil }
	modRootGetwd = func() (string, error) { return externDir, nil }
	t.Setenv(EnvForstGOModRoot, "")

	if got := ForstCompilerModuleRoot(); evalPath(t, got) != evalPath(t, forstMod) {
		t.Fatalf("got %q want %q", got, forstMod)
	}
}
