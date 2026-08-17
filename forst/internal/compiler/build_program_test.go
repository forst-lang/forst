package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/programbuild"
)

func TestBuildNativeProgram_rejectsEmbeddedDisabled(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(ft, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := `{"files":{"include":["**/*.ft"]}}`
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "build")
	c := New(Args{
		Command:  "build",
		FilePath: ft,
		LogLevel: "error",
	}, silentCompilerTestLogger())
	err := c.BuildNativeProgram(outDir, "", "")
	if err == nil || !strings.Contains(err.Error(), "server.embedded") {
		t.Fatalf("BuildNativeProgram() = %v", err)
	}
}

func TestBuildNativeProgram_writesManifestAndBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping native build in short mode")
	}
	forstModule := forstCompilerModuleRoot(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{
  "server": {"embedded": true, "port": "6321"},
  "files": {"include": ["**/*.ft"]}
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	mainFT := `package main

type EchoRequest = {
	message: String
}

type EchoResponse = {
	echo: String,
	timestamp: Int
}

func Echo(input EchoRequest) {
	return {
		echo: input.message,
		timestamp: 42
	}
}

func main() {
	println("embedded invoke listening")
}
`
	ft := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(ft, []byte(mainFT), 0o644); err != nil {
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

	outDir := filepath.Join(dir, "program-build")
	c := New(Args{
		Command:            "build",
		FilePath:           ft,
		ExportStructFields: true,
		LogLevel:           "error",
	}, silentCompilerTestLogger())
	if err := c.BuildNativeProgram(outDir, "", ""); err != nil {
		t.Fatalf("BuildNativeProgram: %v", err)
	}

	manifest, err := programbuild.Load(outDir)
	if err != nil {
		t.Fatalf("programbuild.Load: %v", err)
	}
	if manifest.Kind != programbuild.KindProgram || manifest.Binary != "bin/main" {
		t.Fatalf("manifest = %+v", manifest)
	}
	binPath := filepath.Join(outDir, manifest.Binary)
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("binary missing: %v", err)
	}
}
