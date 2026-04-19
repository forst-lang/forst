package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestRunMain_version(t *testing.T) {
	if code := runMain([]string{"forst", "version"}); code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestRunMain_dev_invalidPort(t *testing.T) {
	tmp := t.TempDir()
	if code := runMain([]string{"forst", "dev", "-port", "notaport", "-root", tmp}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_lsp_invalidPort(t *testing.T) {
	if code := runMain([]string{"forst", "lsp", "-port", "notaport"}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_fmt_list(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "a.ft")
	if err := os.WriteFile(path, []byte("package main  \nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runMain([]string{"forst", "fmt", "-l", path}); code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
}

func TestRunMain_generate_minimal(t *testing.T) {
	tmp := t.TempDir()
	ftPath := filepath.Join(tmp, "sample.ft")
	if err := os.WriteFile(ftPath, []byte(generateTestMinimalValidForst), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runMain([]string{"forst", "generate", ftPath}); code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
}

func TestRunMain_run_missingFileArg(t *testing.T) {
	if code := runMain([]string{"forst", "run"}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_dump_requiresFile(t *testing.T) {
	if code := runMain([]string{"forst", "dump"}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_build_compilesExampleEcho(t *testing.T) {
	echoPath := filepath.Join("..", "..", "..", "examples", "in", "echo.ft")
	if _, err := os.Stat(echoPath); err != nil {
		t.Skip("examples not present:", err)
	}
	if code := runMain([]string{"forst", "build", echoPath}); code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
}

func TestRunMain_fmt_errorsOnMissingPath(t *testing.T) {
	if code := runMain([]string{"forst", "fmt", filepath.Join(t.TempDir(), "missing.ft")}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_generate_errorsOnMissingTarget(t *testing.T) {
	if code := runMain([]string{"forst", "generate", filepath.Join(t.TempDir(), "nope.ft")}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_dump_ok(t *testing.T) {
	tmp := t.TempDir()
	ftPath := filepath.Join(tmp, "sample.ft")
	if err := os.WriteFile(ftPath, []byte("fn main() { return }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	old := dumpCommandStdout
	dumpCommandStdout = &buf
	t.Cleanup(func() { dumpCommandStdout = old })

	code := runMain([]string{"forst", "dump", "--file", ftPath})

	if code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
	if !strings.Contains(buf.String(), "{") {
		t.Fatalf("expected JSON output, got %q", buf.String())
	}
}

func TestRunMain_dev_unknownFlag(t *testing.T) {
	if code := runMain([]string{"forst", "dev", "-not-a-real-flag"}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_dev_invalidRootAbs(t *testing.T) {
	if code := runMain([]string{"forst", "dev", "-root", "a\x00b"}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_dev_pathAbsFails(t *testing.T) {
	orig := pathAbs
	pathAbs = func(string) (string, error) { return "", fmt.Errorf("abs") }
	t.Cleanup(func() { pathAbs = orig })
	if code := runMain([]string{"forst", "dev", "-root", t.TempDir()}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_dev_startReturnsNilExitsZero(t *testing.T) {
	orig := startDevServerFunc
	startDevServerFunc = func(string, *logrus.Logger, string, string, *string) error { return nil }
	t.Cleanup(func() { startDevServerFunc = orig })
	if code := runMain([]string{"forst", "dev", "-root", t.TempDir()}); code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
}

func TestRunMain_lsp_unknownFlag(t *testing.T) {
	if code := runMain([]string{"forst", "lsp", "-bogus"}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_lsp_startReturnsNilExitsZero(t *testing.T) {
	orig := startLSPFunc
	startLSPFunc = func(string, *logrus.Logger) error { return nil }
	t.Cleanup(func() { startLSPFunc = orig })
	if code := runMain([]string{"forst", "lsp"}); code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
}

func TestRunMain_dump_unknownFlag(t *testing.T) {
	if code := runMain([]string{"forst", "dump", "-nope"}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_watch_missingFile_returnsError(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.go")
	missing := filepath.Join(tmp, "missing.ft")
	if code := runMain([]string{"forst", "run", "-watch", "-o", out, missing}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_build_invalidForst(t *testing.T) {
	ft := filepath.Join(t.TempDir(), "bad.ft")
	if err := os.WriteFile(ft, []byte("not forst {{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runMain([]string{"forst", "build", ft}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_build_createTempFails(t *testing.T) {
	echoPath := filepath.Join("..", "..", "..", "examples", "in", "echo.ft")
	if _, err := os.Stat(echoPath); err != nil {
		t.Skip("examples not present:", err)
	}
	orig := createTempOutputFileFn
	createTempOutputFileFn = func(string) (string, error) { return "", fmt.Errorf("temp") }
	t.Cleanup(func() { createTempOutputFileFn = orig })
	if code := runMain([]string{"forst", "build", echoPath}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_run_goProgramFails(t *testing.T) {
	echoPath := filepath.Join("..", "..", "..", "examples", "in", "echo.ft")
	if _, err := os.Stat(echoPath); err != nil {
		t.Skip("examples not present:", err)
	}
	orig := runGoProgramFn
	runGoProgramFn = func(string) error { return fmt.Errorf("no run") }
	t.Cleanup(func() { runGoProgramFn = orig })
	if code := runMain([]string{"forst", "run", echoPath}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestRunMain_run_exitsZeroWhenGoRunSucceeds(t *testing.T) {
	echoPath := filepath.Join("..", "..", "..", "examples", "in", "echo.ft")
	if _, err := os.Stat(echoPath); err != nil {
		t.Skip("examples not present:", err)
	}
	orig := runGoProgramFn
	runGoProgramFn = func(string) error { return nil }
	t.Cleanup(func() { runGoProgramFn = orig })
	if code := runMain([]string{"forst", "run", echoPath}); code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
}
