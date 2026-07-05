package testrunner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverPackages_readDirErrorAfterWalk(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses chmod 000")
	}
	root := t.TempDir()
	secret := filepath.Join(root, "secret")
	if err := os.Mkdir(secret, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(secret, "a_test.ft"), "package secret\n")
	link := filepath.Join(root, "link")
	if err := os.Symlink(secret, link); err != nil {
		t.Skip("symlinks unavailable")
	}
	if err := os.Chmod(secret, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(secret, 0o755) })
	if _, err := DiscoverPackages(root, nil); err == nil {
		t.Fatal("expected readdir error via symlink to unreadable dir")
	}
}

func TestDiscoverPackages_skipsSubdirectoriesInPackageDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pkg", "nested", "ignored.ft"), "package pkg\n")
	writeFile(t, filepath.Join(root, "pkg", "a_test.ft"), "package pkg\n")
	pkgs, err := DiscoverPackages(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %#v", pkgs)
	}
}
