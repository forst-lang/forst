package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestWatchFile_rejectsPackageRoot(t *testing.T) {
	c := New(Args{
		FilePath:    filepath.Join(t.TempDir(), "entry.ft"),
		PackageRoot: t.TempDir(),
	}, nil)
	err := c.WatchFile()
	if err == nil {
		t.Fatal("expected -watch with -root error")
	}
}

func TestWatchFile_missingWatchedFile_returnsBeforeEventLoop(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "not-created.ft")
	out := filepath.Join(dir, "out.go")

	c := New(Args{
		FilePath:   missing,
		OutputPath: out,
		LogLevel:   "error",
	}, nil)

	err := c.WatchFile()
	if err == nil {
		t.Fatal("expected error from fsnotify.Add on missing path")
	}
}

func TestCompileAndRunOnce_compileFailureDoesNotPanic(t *testing.T) {
	c := New(Args{
		Command:  "build",
		FilePath: filepath.Join(t.TempDir(), "missing.ft"),
		LogLevel: "error",
	}, nil)
	// should log and return without panic
	c.compileAndRunOnce()
}

func TestCompileAndRunOnce_successInvokesRunGoProgramWithOutputPath(t *testing.T) {
	origRun := runGoProgramForWatch
	t.Cleanup(func() { runGoProgramForWatch = origRun })

	dir := t.TempDir()
	ftPath := filepath.Join(dir, "ok.ft")
	if err := os.WriteFile(ftPath, []byte(`package main

func main() {
	println("ok")
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "compiled.go")

	var gotPath string
	runGoProgramForWatch = func(p string) error {
		gotPath = p
		return nil
	}

	c := New(Args{
		Command:    "build",
		FilePath:   ftPath,
		OutputPath: outPath,
		LogLevel:   "error",
	}, nil)
	c.compileAndRunOnce()

	if gotPath != outPath {
		t.Fatalf("expected go run on %q, got %q", outPath, gotPath)
	}
}

func TestRunCompiledOutput_usesExplicitOutputPath(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "main.go")
	goCode := "package main\nfunc main() {}\n"
	if err := os.WriteFile(goFile, []byte(goCode), 0644); err != nil {
		t.Fatal(err)
	}

	c := New(Args{
		OutputPath: goFile,
	}, nil)
	if err := c.runCompiledOutput("ignored because output path is set"); err != nil {
		t.Fatalf("runCompiledOutput: %v", err)
	}
}

func TestValidateWatchConfig_acceptsNoPackageRoot(t *testing.T) {
	c := New(Args{FilePath: filepath.Join(t.TempDir(), "entry.ft")}, nil)
	if err := c.validateWatchConfig(); err != nil {
		t.Fatalf("validateWatchConfig unexpected error: %v", err)
	}
}

func TestResolveOutputPathForRun_prefersExplicitPath(t *testing.T) {
	c := New(Args{OutputPath: "/tmp/out.go"}, nil)
	got, err := c.resolveOutputPathForRun("ignored")
	if err != nil {
		t.Fatalf("resolveOutputPathForRun: %v", err)
	}
	if got != "/tmp/out.go" {
		t.Fatalf("expected explicit output path, got %q", got)
	}
}

func TestRunCompiledOutput_tempCreationFailureReturnsError(t *testing.T) {
	origCreate := createTempOutputFileForWatch
	origRun := runGoProgramForWatch
	t.Cleanup(func() {
		createTempOutputFileForWatch = origCreate
		runGoProgramForWatch = origRun
	})

	createTempOutputFileForWatch = func(string) (string, error) {
		return "", fmt.Errorf("temp create failed")
	}
	runGoProgramForWatch = func(string) error {
		t.Fatal("runGoProgramForWatch should not be called when temp creation fails")
		return nil
	}

	c := New(Args{}, nil)
	err := c.runCompiledOutput("package main\nfunc main(){}\n")
	if err == nil {
		t.Fatal("expected temp creation error")
	}
	if err.Error() != "temp create failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCompiledOutput_runFailureReturnsError(t *testing.T) {
	origCreate := createTempOutputFileForWatch
	origRun := runGoProgramForWatch
	t.Cleanup(func() {
		createTempOutputFileForWatch = origCreate
		runGoProgramForWatch = origRun
	})

	createTempOutputFileForWatch = func(string) (string, error) {
		return "/tmp/generated.go", nil
	}
	runGoProgramForWatch = func(string) error {
		return fmt.Errorf("run failed")
	}

	c := New(Args{}, nil)
	err := c.runCompiledOutput("package main\nfunc main(){}\n")
	if err == nil {
		t.Fatal("expected run failure")
	}
	if err.Error() != "run failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}
