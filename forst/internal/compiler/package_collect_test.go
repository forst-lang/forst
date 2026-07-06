package compiler

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func silentCompilerTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

func TestCollectSamePackageFtPaths_filtersAndSortsAndSkipsUnparseable(t *testing.T) {
	root := t.TempDir()
	logger := silentCompilerTestLogger()

	entryPath := filepath.Join(root, "entry.ft")
	if err := os.WriteFile(entryPath, []byte(`package demo

func Entry(): String {
	return "ok"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	samePackagePath := filepath.Join(root, "b.ft")
	if err := os.WriteFile(samePackagePath, []byte(`package demo

func B(): String {
	return "b"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	otherPackagePath := filepath.Join(root, "a.ft")
	if err := os.WriteFile(otherPackagePath, []byte(`package other

func A(): String {
	return "a"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	unparseablePath := filepath.Join(root, "z.ft")
	if err := os.WriteFile(unparseablePath, []byte("@@@ invalid @@@"), 0o644); err != nil {
		t.Fatal(err)
	}
	testPath := filepath.Join(root, "demo_test.ft")
	if err := os.WriteFile(testPath, []byte(`package demo

import "testing"

func TestDemo(t *testing.T) {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := collectSamePackageFtPaths(logger, root, entryPath)
	if err != nil {
		t.Fatalf("collectSamePackageFtPaths: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 same-package paths (entry + b, excluding demo_test.ft), got %d: %+v", len(got), got)
	}
	for _, p := range got {
		if strings.HasSuffix(p, "_test.ft") {
			t.Fatalf("collectSamePackageFtPaths must skip *_test.ft, got %q", p)
		}
	}
	if got[0] != filepath.Join(root, "b.ft") || got[1] != entryPath {
		t.Fatalf("expected sorted same-package paths, got %+v", got)
	}
}

func TestCollectSamePackageFtPaths_parseEntryError(t *testing.T) {
	root := t.TempDir()
	logger := silentCompilerTestLogger()

	entryPath := filepath.Join(root, "entry.ft")
	if err := os.WriteFile(entryPath, []byte(`package demo

func Entry(): String {
	return "ok"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "other.ft"), []byte(`package other

func Other(): String {
	return "x"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(entryPath); err != nil {
		t.Fatal(err)
	}

	_, err := collectSamePackageFtPaths(logger, root, entryPath)
	if err == nil {
		t.Fatal("expected parse entry file error when entry is missing")
	}
	if !strings.Contains(err.Error(), "parse entry file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadMergedPackageAST_entryOutsideRootReturnsError(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	entryPath := filepath.Join(outsideDir, "entry.ft")
	if err := os.WriteFile(entryPath, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	compiler := New(Args{
		Command:     "build",
		FilePath:    entryPath,
		PackageRoot: root,
		LogLevel:    "error",
	}, silentCompilerTestLogger())

	_, err := compiler.loadMergedPackageAST()
	if err == nil {
		t.Fatal("expected error when entry file is outside root")
	}
	if !strings.Contains(err.Error(), "is not under -root") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollectSamePackageFtPaths_openRootError(t *testing.T) {
	logger := silentCompilerTestLogger()
	_, err := collectSamePackageFtPaths(logger, filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "entry.ft"))
	if err == nil || !strings.Contains(err.Error(), "open package root") {
		t.Fatalf("err = %v", err)
	}
}

func TestCollectSamePackageFtPaths_walkError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses chmod")
	}
	root := t.TempDir()
	logger := silentCompilerTestLogger()
	entryPath := filepath.Join(root, "entry.ft")
	if err := os.WriteFile(entryPath, []byte(`package demo

func Entry(): String { return "ok" }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(root, "blocked")
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blocked, "peer.ft"), []byte(`package demo

func Peer(): String { return "p" }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blocked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })

	_, err := collectSamePackageFtPaths(logger, root, entryPath)
	if err == nil {
		t.Fatal("expected walk error from unreadable subdirectory")
	}
}

func TestLoadMergedPackageAST_mergesSamePackage(t *testing.T) {
	root := t.TempDir()
	entryPath := filepath.Join(root, "entry.ft")
	if err := os.WriteFile(entryPath, []byte(`package demo

func Entry(): String { return "ok" }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "peer.ft"), []byte(`package demo

func Peer(): String { return "p" }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New(Args{
		Command:     "build",
		FilePath:    entryPath,
		PackageRoot: root,
		LogLevel:    "error",
	}, silentCompilerTestLogger())
	nodes, err := c.loadMergedPackageAST()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("loadMergedPackageAST: err=%v len=%d", err, len(nodes))
	}
}
