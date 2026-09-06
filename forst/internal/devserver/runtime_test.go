package devserver

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/compiler"
	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

func writeEntry(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveEntry_cliOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, "forst/main.ft", "package main\n")
	writeEntry(t, dir, "main.ft", "package main\n")
	cfg := &ftconfig.Config{Dev: ftconfig.DevConfig{Entry: "main.ft"}}
	got, err := ResolveEntry(dir, cfg, "forst/main.ft")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(filepath.Dir(got)) != "forst" {
		t.Fatalf("want forst/main.ft, got %q", got)
	}
}

func TestResolveEntry_configOverridesConvention(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, "main.ft", "package main\n")
	cfg := &ftconfig.Config{Dev: ftconfig.DevConfig{Entry: "main.ft"}}
	got, err := ResolveEntry(dir, cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "main.ft" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveEntry_conventionForstMain(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, "forst/main.ft", "package main\n")
	got, err := ResolveEntry(dir, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, filepath.Join("forst", "main.ft")) {
		t.Fatalf("got %q", got)
	}
}

func TestResolveEntry_conventionRootMain(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, "main.ft", "package main\n")
	got, err := ResolveEntry(dir, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "main.ft" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveEntry_missingReturnsTriedPaths(t *testing.T) {
	dir := t.TempDir()
	_, err := ResolveEntry(dir, nil, "")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "tried:") {
		t.Fatalf("missing tried list: %v", err)
	}
	if !strings.Contains(msg, "forst/main.ft") || !strings.Contains(msg, "main.ft") {
		t.Fatalf("missing convention paths: %v", err)
	}
}

func TestRunRuntimeDev_callsCompileAndRun(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(entry, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var gotArgs compiler.Args
	var runBoundary string
	var runOutput string
	log := logrus.New()
	log.SetOutput(os.Stderr)
	deps := RuntimeRunDeps{
		NewCompiler: func(args compiler.Args, _ *logrus.Logger) *compiler.Compiler {
			gotArgs = args
			return compiler.New(args, log)
		},
		CreateOutput: func(main, _, _ string, _ map[string]string, _ map[string]string, boundary string) (string, error) {
			if main == "" {
				return "", errors.New("empty main")
			}
			return filepath.Join(boundary, ".forst", "run", "test", "main.go"), nil
		},
		RunProgram: func(outputPath, boundaryRoot string) error {
			runOutput = outputPath
			runBoundary = boundaryRoot
			return nil
		},
	}
	cfg := &ftconfig.Config{Compiler: ftconfig.CompilerConfig{ExportStructFields: true}}
	if err := RunRuntimeDev(log, dir, entry, cfg, deps); err != nil {
		t.Fatal(err)
	}
	if gotArgs.FilePath != entry || gotArgs.PackageRoot != dir {
		t.Fatalf("compile args: %+v", gotArgs)
	}
	if runBoundary != dir || runOutput == "" {
		t.Fatalf("run: boundary=%q output=%q", runBoundary, runOutput)
	}
}

func TestRunRuntimeDev_compileErrorPropagates(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	deps := RuntimeRunDeps{
		NewCompiler: func(args compiler.Args, l *logrus.Logger) *compiler.Compiler {
			return compiler.New(args, l)
		},
		CreateOutput: func(string, string, string, map[string]string, map[string]string, string) (string, error) {
			return "", errors.New("should not create output")
		},
		RunProgram: func(string, string) error {
			return errors.New("should not run")
		},
	}
	err := RunRuntimeDev(log, t.TempDir(), filepath.Join(t.TempDir(), "missing.ft"), nil, deps)
	if err == nil {
		t.Fatal("expected compile error")
	}
}

func TestCompileRuntimeOutput_usesOverlayReplacesWhenPresent(t *testing.T) {
	root := t.TempDir()
	consumer := filepath.Join(root, "consumer")
	lib := filepath.Join(root, "acme-lib")
	writeEntry(t, lib, "go.mod", "module github.com/acme/lib\n\ngo 1.26.0\n")
	writeEntry(t, lib, "add.ft", "package lib\n\nfunc Add(a Int, b Int) {\n\treturn a + b\n}\n")
	writeEntry(t, consumer, "go.mod", "module testmod\n\ngo 1.26.0\n\nrequire github.com/acme/lib v0.0.0\n\nreplace github.com/acme/lib => ../acme-lib\n")
	writeEntry(t, consumer, "main.ft", "package main\n\nimport \"github.com/acme/lib\"\n\nfunc main() {\n\t_ = lib.Add(1, 2)\n}\n")
	entry := filepath.Join(consumer, "main.ft")
	log := logrus.New()
	log.SetOutput(io.Discard)
	createCalled := false
	deps := RuntimeRunDeps{
		NewCompiler: func(args compiler.Args, l *logrus.Logger) *compiler.Compiler {
			return compiler.New(args, l)
		},
		CreateOutput: func(string, string, string, map[string]string, map[string]string, string) (string, error) {
			createCalled = true
			return "", errors.New("CreateOutput must not be used when overlays exist")
		},
	}
	out, _, err := compileRuntimeOutput(log, consumer, entry, &ftconfig.Config{}, deps, false)
	if err != nil {
		t.Fatalf("compileRuntimeOutput: %v", err)
	}
	if createCalled {
		t.Fatal("expected overlay-aware sandbox path, not CreateOutput")
	}
	goMod, err := os.ReadFile(filepath.Join(filepath.Dir(out), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goMod), "replace github.com/acme/lib =>") {
		t.Fatalf("sandbox go.mod missing overlay replace:\n%s", goMod)
	}
	overlayRoot := filepath.Join(consumer, ".forst", "overlay")
	entries, err := os.ReadDir(overlayRoot)
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected overlay dir under %s: %v", overlayRoot, err)
	}
}
