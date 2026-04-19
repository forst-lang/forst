package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	done := make(chan struct{})
	var out []byte
	go func() {
		defer close(done)
		out, _ = io.ReadAll(r)
		_ = r.Close()
	}()

	code := runMain([]string{"forst", "dump", "--file", ftPath})
	_ = w.Close()
	<-done

	if code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
	if !strings.Contains(string(out), "{") {
		t.Fatalf("expected JSON output, got %q", string(out))
	}
}
