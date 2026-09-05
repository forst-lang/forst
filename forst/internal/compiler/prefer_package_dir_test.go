package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreferPackageDirRunEmit_noFtconfigLeavesOutputEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ft")
	c := New(Args{
		Command:  "run",
		FilePath: entry,
		LogLevel: "error",
	}, silentCompilerTestLogger())
	c.PreferPackageDirRunEmit()
	if c.Args.OutputPath != "" {
		t.Fatalf("OutputPath = %q, want empty (sandbox)", c.Args.OutputPath)
	}
}

func TestPreferPackageDirRunEmit_usesConfiguredGenerateGoOut(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(entry, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := `{
  "generate": {
    "go": {
      "entry": "./main.ft",
      "out": "./main.gen.go"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New(Args{
		Command:  "run",
		FilePath: entry,
		LogLevel: "error",
	}, silentCompilerTestLogger())
	c.PreferPackageDirRunEmit()
	want := filepath.Join(dir, "main.gen.go")
	if c.Args.OutputPath != want {
		t.Fatalf("OutputPath = %q, want %q", c.Args.OutputPath, want)
	}
}

func TestPreferPackageDirRunEmit_skipsWhenOutputSet(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ft")
	c := New(Args{
		Command:    "run",
		FilePath:   entry,
		OutputPath: filepath.Join(dir, "custom.go"),
		LogLevel:   "error",
	}, silentCompilerTestLogger())
	c.PreferPackageDirRunEmit()
	if !strings.HasSuffix(c.Args.OutputPath, "custom.go") {
		t.Fatalf("should keep explicit OutputPath, got %q", c.Args.OutputPath)
	}
}

func TestPreferPackageDirRunEmit_skipsNonRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	c := New(Args{
		Command:  "generate",
		FilePath: filepath.Join(dir, "main.ft"),
		LogLevel: "error",
	}, silentCompilerTestLogger())
	c.PreferPackageDirRunEmit()
	if c.Args.OutputPath != "" {
		t.Fatalf("generate must not set package-dir emit, got %q", c.Args.OutputPath)
	}
}

func TestDefaultPackageGoOut_stemFallbackForBuild(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	entry := filepath.Join(dir, "app.ft")
	c := New(Args{
		Command:  "build",
		FilePath: entry,
		LogLevel: "error",
	}, silentCompilerTestLogger())
	got := c.defaultPackageGoOut()
	want := filepath.Join(dir, "app.gen.go")
	if got != want {
		t.Fatalf("defaultPackageGoOut = %q, want %q", got, want)
	}
}
