package testrunner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverPackages_walkPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can read chmod 000 directories")
	}
	root := t.TempDir()
	secret := filepath.Join(root, "secret")
	if err := os.Mkdir(secret, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(secret, 0o755) })
	if _, err := DiscoverPackages(root, nil); err == nil {
		t.Fatal("expected walk error")
	}
}

func TestDiscoverPackages_explicitAbsolutePath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	pkgDir := filepath.Join(root, "svc")
	writeFile(t, filepath.Join(pkgDir, "svc.ft"), "package svc\n")
	writeFile(t, filepath.Join(pkgDir, "svc_test.ft"), "package svc\n")

	pkgs, err := DiscoverPackages(root, []string{pkgDir})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].RelPath != "svc" {
		t.Fatalf("got %#v", pkgs)
	}
}

func TestDiscoverPackages_readDirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses chmod 000")
	}
	root := t.TempDir()
	pkgDir := filepath.Join(root, "locked")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(pkgDir, "a_test.ft"), "package locked\n")
	if err := os.Chmod(pkgDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(pkgDir, 0o755) })
	if _, err := DiscoverPackages(root, []string{"locked"}); err == nil {
		t.Fatal("expected read dir error")
	}
}
