package compiler

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPreferPackageDirRunEmit_setsGenBesideEntry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ft")
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
