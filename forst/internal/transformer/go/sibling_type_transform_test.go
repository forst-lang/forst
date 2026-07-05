package transformergo

import (
	"bytes"
	"go/format"
	"go/token"
	"os"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/modulecheck"
	"forst/internal/testmod"
)

func TestTransformForstSiblingQualifiedParamType(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	testmod.WriteGoMod(t, dir, "xpkg")
	runctxDir := dir + "/runctx"
	exportDir := dir + "/export"
	for _, d := range []string{runctxDir, exportDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeSiblingTestFile(t, runctxDir+"/runctx.ft", `package runctx

type RunContext = {
  runId: String,
}
`)
	writeSiblingTestFile(t, exportDir+"/manifest.ft", `package export

import "xpkg/runctx"

func WriteManifest(ctx runctx.RunContext): Error {
  return nil
}
`)

	modResult, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	exportTC := modResult.ForstPackageTypeChecker("export")
	if exportTC == nil {
		t.Fatal("missing export typechecker")
	}
	merged, _, err := forstpkg.ParseAndMergePackage(nil, []string{exportDir + "/manifest.ft"})
	if err != nil {
		t.Fatalf("parse export: %v", err)
	}
	log := ast.SetupTestLogger(nil)
	log.SetOutput(bytes.NewBuffer(nil))
	tr := New(exportTC, log, true)
	tr.SetModuleResult(modResult)
	goFile, err := tr.TransformForstFileToGo(merged)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), goFile); err != nil {
		t.Fatalf("format: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "func WriteManifest(ctx runctx.RunContext)") {
		t.Fatalf("expected runctx.RunContext param, got:\n%s", out)
	}
	if strings.Contains(out, "T_") {
		t.Fatalf("expected no hash types, got:\n%s", out)
	}
}

func writeSiblingTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
