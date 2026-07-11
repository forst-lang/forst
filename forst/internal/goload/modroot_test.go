package goload

import (
	"os"
	"path/filepath"
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

	if got := ForstCompilerModuleRoot(); got != forstMod {
		t.Fatalf("got %q want %q", got, forstMod)
	}
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

	if got := ForstCompilerModuleRoot(); got != moduleDir {
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
