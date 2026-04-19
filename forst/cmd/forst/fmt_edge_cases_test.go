package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/internal/printer"
)

func TestRunFmtCommand_skipsUnchangedFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeFmtTestFile(t, dir, "a.ft", "package main\nfunc main() {\n}\n", 0o644)
	formatted := printer.FormatDocument("package main\nfunc main() {\n}\n", path, 8, false, testFmtLogger())
	if err := os.WriteFile(path, []byte(formatted), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runFmt(t, "-l", path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected no list output for unchanged file, got %q", out)
	}
}

func TestRunFmtCommand_readError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based unreadable file not portable on Windows")
	}
	dir := t.TempDir()
	path := writeFmtTestFile(t, dir, "ro.ft", "package main\nfunc main() {}\n", 0o644)
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	err := runFmtCommand([]string{path}, testFmtLogger(), ioDiscard{})
	if err == nil {
		t.Fatal("expected error when source file is unreadable")
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func TestRunFmtCommand_unknownFlag(t *testing.T) {
	err := runFmtCommand([]string{"-not-a-fmt-flag"}, testFmtLogger(), ioDiscard{})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestCollectFtPaths_walkError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based walk error not portable on Windows")
	}
	root := t.TempDir()
	sub := filepath.Join(root, "inaccessible")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.ft"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sub, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sub, 0o755) })

	_, err := collectFtPaths([]string{root})
	if err == nil {
		t.Fatal("expected walk error")
	}
}

func TestRunFmtCommand_writeError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod read-only file not portable on Windows")
	}
	dir := t.TempDir()
	path := writeFmtTestFile(t, dir, "a.ft", "package main  \nfunc main() {\n}\n", 0o644)
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	err := runFmtCommand([]string{path}, testFmtLogger(), ioDiscard{})
	if err == nil {
		t.Fatal("expected error when file cannot be written")
	}
}
