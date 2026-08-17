package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"forst/internal/compiler"
	"forst/internal/programbuild"
)

func forstCompilerModuleRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

type embeddedBuildFixtureOpts struct {
	ftconfig    string
	mainFT      string
	entryName   string
	packageRoot string
}

type embeddedBuildFixture struct {
	dir   string
	entry string
	c     *compiler.Compiler
}

func writeEmbeddedBuildFixture(t *testing.T, opts embeddedBuildFixtureOpts) embeddedBuildFixture {
	t.Helper()
	forstModule := forstCompilerModuleRoot(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(opts.ftconfig), 0o644); err != nil {
		t.Fatal(err)
	}
	entryName := opts.entryName
	if entryName == "" {
		entryName = "main.ft"
	}
	entry := filepath.Join(dir, entryName)
	if err := os.WriteFile(entry, []byte(opts.mainFT), 0o644); err != nil {
		t.Fatal(err)
	}
	forstGomod := filepath.Join(dir, ".forst-gomod")
	if err := os.MkdirAll(forstGomod, 0o755); err != nil {
		t.Fatal(err)
	}
	goMod := "module example.com/test\n\ngo 1.26.0\n\nrequire forst v0.0.0\n\nreplace forst => " + forstModule + "\n"
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	c := compiler.New(compiler.Args{
		Command:            "build",
		FilePath:           entry,
		PackageRoot:        opts.packageRoot,
		ExportStructFields: true,
		LogLevel:           "error",
	}, exampleTestLogger())
	return embeddedBuildFixture{dir: dir, entry: entry, c: c}
}

func buildProgram(t *testing.T, c *compiler.Compiler, outDir string) programbuild.ProgramManifest {
	t.Helper()
	if err := c.BuildNativeProgram(outDir, "", ""); err != nil {
		t.Fatalf("BuildNativeProgram: %v", err)
	}
	manifest, err := programbuild.Load(outDir)
	if err != nil {
		t.Fatalf("programbuild.Load: %v", err)
	}
	return manifest
}
