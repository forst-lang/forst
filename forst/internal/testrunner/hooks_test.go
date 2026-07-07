package testrunner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ast"
	transformer_go "forst/internal/transformer/go"

	goast "go/ast"
)

func TestWriteGeneratedTestAndRun_execGoTestError(t *testing.T) {
	orig := execGoTest
	t.Cleanup(func() { execGoTest = orig })
	execGoTest = func(*exec.Cmd) error {
		return errors.New("go test unavailable")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module m\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	code, err := writeGeneratedTestAndRun(PackageUnderTest{
		Dir:     pkgDir,
		RelPath: "pkg",
	}, "package pkg\n", nil, testLog(t))
	if err == nil || code != ExitFailure {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !strings.Contains(err.Error(), "go test") {
		t.Fatalf("err = %v", err)
	}
}

func TestRun_filepathAbsError(t *testing.T) {
	orig := filepathAbs
	t.Cleanup(func() { filepathAbs = orig })
	filepathAbs = func(string) (string, error) {
		return "", errors.New("abs failed")
	}
	code, err := Run(Options{ModuleRoot: ".", Log: testLog(t)})
	if err == nil || code != ExitError {
		t.Fatalf("code=%d err=%v", code, err)
	}
}

func TestEmitPackageGo_transformError(t *testing.T) {
	orig := transformForstFileToGo
	t.Cleanup(func() { transformForstFileToGo = orig })
	transformForstFileToGo = func(*transformer_go.Transformer, []ast.Node) (*goast.File, error) {
		return nil, errors.New("transform failed")
	}

	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "ok")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(pkgDir, "ok.ft")
	ft := filepath.Join(pkgDir, "ok_test.ft")
	if err := os.WriteFile(lib, []byte(`package ok

func ok(): Int { return 1 }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ft, []byte(`package ok

import "testing"

func TestOk(t *testing.T) {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := PackageUnderTest{
		Dir:     pkgDir,
		RelPath: "ok",
		FtPaths: []string{lib, ft},
	}
	_, err := emitPackageGo(dir, pkg, nil, EmitOptions{}, testLog(t))
	if err == nil || !strings.Contains(err.Error(), "transform") {
		t.Fatalf("err = %v", err)
	}
}

func TestDiscoverPackages_explicitPathAbsError(t *testing.T) {
	orig := filepathAbs
	t.Cleanup(func() { filepathAbs = orig })
	filepathAbs = func(string) (string, error) {
		return "", errors.New("abs failed")
	}
	_, err := DiscoverPackages(t.TempDir(), []string{"pkg"})
	if err == nil {
		t.Fatal("expected abs error")
	}
}

func TestDiscoverPackages_relPathErrorInLoop(t *testing.T) {
	orig := filepathRelDiscover
	t.Cleanup(func() { filepathRelDiscover = orig })
	filepathRelDiscover = func(string, string) (string, error) {
		return "", errors.New("rel failed")
	}
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pkg", "a_test.ft"), "package pkg\n")
	if _, err := DiscoverPackages(root, nil); err == nil {
		t.Fatal("expected rel error")
	}
}

func TestDiscoverPackages_readDirErrorInPackageLoop(t *testing.T) {
	orig := readDirFn
	t.Cleanup(func() { readDirFn = orig })
	root := t.TempDir()
	pkgDir := filepath.Join(root, "pkg")
	writeFile(t, filepath.Join(pkgDir, "a_test.ft"), "package pkg\n")

	readDirFn = func(name string) ([]os.DirEntry, error) {
		if name == pkgDir {
			return nil, fmt.Errorf("read failed")
		}
		return os.ReadDir(name)
	}
	if _, err := DiscoverPackages(root, nil); err == nil {
		t.Fatal("expected readdir error in package loop")
	}
}

func TestDiscoverPackages_skipsSubdirectoryEntriesInLoop(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "pkg")
	writeFile(t, filepath.Join(pkgDir, "a_test.ft"), "package pkg\n")
	if err := os.Mkdir(filepath.Join(pkgDir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	pkgs, err := DiscoverPackages(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || len(pkgs[0].TestPaths) != 1 {
		t.Fatalf("got %#v", pkgs)
	}
}

func TestDiscoverPackages_skipsDirWhenSecondReadHasNoTestFiles(t *testing.T) {
	orig := readDirFn
	t.Cleanup(func() { readDirFn = orig })
	root := t.TempDir()
	pkgDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var calls int
	readDirFn = func(name string) ([]os.DirEntry, error) {
		if name != pkgDir {
			return os.ReadDir(name)
		}
		calls++
		if calls == 1 {
			// explicit-path style: first read finds a test file during walk... walk uses WalkDir not readDirFn
			return os.ReadDir(name)
		}
		// Simulate dir listed from walk but second enumeration has only non-test .ft files.
		return []os.DirEntry{fakeDirEntry{name: "plain.ft", isDir: false}}, nil
	}

	// Seed testDirs via walk by creating a test file, then stub second read.
	writeFile(t, filepath.Join(pkgDir, "a_test.ft"), "package pkg\n")
	// Override: after walk adds pkgDir, loop read returns no test files.
	calls = 0
	readDirFn = func(name string) ([]os.DirEntry, error) {
		if name == pkgDir {
			return []os.DirEntry{fakeDirEntry{name: "plain.ft", isDir: false}}, nil
		}
		return os.ReadDir(name)
	}
	pkgs, err := DiscoverPackages(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected empty package list, got %#v", pkgs)
	}
}

func TestRelPath_fallbackWhenRelFails(t *testing.T) {
	t.Parallel()
	orig := filepathRel
	t.Cleanup(func() { filepathRel = orig })
	filepathRel = func(string, string) (string, error) {
		return "", errors.New("rel failed")
	}
	dir := filepath.Join(t.TempDir(), "auth")
	if got := relPath(t.TempDir(), dir); got != dir {
		t.Fatalf("relPath = %q, want %q", got, dir)
	}
}

type fakeDirEntry struct {
	name  string
	isDir bool
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return f.isDir }
func (f fakeDirEntry) Type() os.FileMode          { return 0 }
func (f fakeDirEntry) Info() (os.FileInfo, error) { return nil, errors.New("not implemented") }
