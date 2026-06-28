package safefs

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRoot_readInside(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.ft")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	if got := r.AbsRoot(); got != filepath.Clean(dir) {
		t.Fatalf("AbsRoot=%q want %q", got, filepath.Clean(dir))
	}

	b, err := r.ReadFile("a.ft")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(b) != "package main\n" {
		t.Fatalf("got %q", b)
	}

	info, err := r.Stat("a.ft")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Name() != "a.ft" {
		t.Fatalf("Stat name=%q", info.Name())
	}
}

func TestOpenRoot_notDirectory(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := OpenRoot(file)
	if err == nil {
		t.Fatal("expected error opening file as root")
	}
}

func TestOpenRoot_escapeDotDot(t *testing.T) {
	dir := t.TempDir()
	r, err := OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	_, err = r.ReadFile("../outside")
	if err == nil {
		t.Fatal("expected error reading outside root via ..")
	}
}

func TestRelPath_insideAndOutside(t *testing.T) {
	dir := t.TempDir()
	r, err := OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	inside := filepath.Join(dir, "sub", "file.ft")
	if err := os.MkdirAll(filepath.Dir(inside), 0o755); err != nil {
		t.Fatal(err)
	}
	rel, err := r.RelPath(inside)
	if err != nil {
		t.Fatalf("RelPath inside: %v", err)
	}
	if rel != "sub/file.ft" && rel != "sub\\file.ft" {
		// RelPath returns ToSlash
		if filepath.ToSlash(rel) != "sub/file.ft" {
			t.Fatalf("rel=%q", rel)
		}
	}

	outside := filepath.Join(filepath.Dir(dir), "outside.ft")
	_, err = r.RelPath(outside)
	if err == nil {
		t.Fatal("expected error for path outside root")
	}
}

func TestWalkDir_collectsFt(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.ft"), []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.go"), []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	var ftPaths []string
	err = fs.WalkDir(r.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".ft") {
			ftPaths = append(ftPaths, r.AbsPath(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
	if len(ftPaths) != 1 {
		t.Fatalf("expected 1 .ft file, got %v", ftPaths)
	}
	want := filepath.Join(dir, "pkg", "a.ft")
	if ftPaths[0] != want {
		t.Fatalf("AbsPath via walk: got %q want %q", ftPaths[0], want)
	}
}

func TestSymlinkEscape_unix(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("symlink escape test on unix")
	}
	root := t.TempDir()
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "secret")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(outsideFile, link); err != nil {
		t.Skip("symlink not permitted")
	}

	r, err := OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	_, err = r.ReadFile("link")
	if err == nil {
		t.Fatal("expected symlink escape to be blocked")
	}
}
