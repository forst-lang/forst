package testrunner

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIsTestForstFile(t *testing.T) {
	t.Parallel()
	if !IsTestForstFile("auth_test.ft") {
		t.Fatal("expected *_test.ft")
	}
	if IsTestForstFile("main.ft") {
		t.Fatal("main.ft is not a test file")
	}
}

func TestDiscoverPackages_walksModuleAndSkipsDotDirs(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pkg", "a.ft"), "package pkg\n")
	writeFile(t, filepath.Join(root, "pkg", "a_test.ft"), "package pkg\n")
	writeFile(t, filepath.Join(root, ".hidden", "x_test.ft"), "package hidden\n")

	pkgs, err := DiscoverPackages(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1 (skip .hidden)", len(pkgs))
	}
	if pkgs[0].RelPath != "pkg" {
		t.Fatalf("RelPath = %q", pkgs[0].RelPath)
	}
	if len(pkgs[0].TestPaths) != 1 {
		t.Fatalf("TestPaths = %v", pkgs[0].TestPaths)
	}
}

func TestDiscoverPackages_explicitPathToPackageDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	pkgDir := filepath.Join(root, "auth")
	writeFile(t, filepath.Join(pkgDir, "svc.ft"), "package auth\n")
	writeFile(t, filepath.Join(pkgDir, "svc_test.ft"), "package auth\n")

	pkgs, err := DiscoverPackages(root, []string{"auth"})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].RelPath != "auth" {
		t.Fatalf("got %#v", pkgs)
	}
}

func TestDiscoverPackages_explicitPathToTestFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	testPath := filepath.Join(root, "demo", "demo_test.ft")
	writeFile(t, testPath, "package demo\n")

	pkgs, err := DiscoverPackages(root, []string{filepath.Join("demo", "demo_test.ft")})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %#v", pkgs)
	}
}

func TestDiscoverPackages_noTestsErrors(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "only.ft"), "package main\n")
	if _, err := DiscoverPackages(root, nil); err == nil {
		t.Fatal("expected error when no *_test.ft")
	}
}

func TestDiscoverPackages_dirWithoutTestsErrors(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "plain.ft"), "package main\n")
	if _, err := DiscoverPackages(root, []string{"."}); err == nil {
		t.Fatal("expected error for dir without tests")
	}
}
