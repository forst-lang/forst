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
	workspaceDir := dir + "/workspace"
	reportsDir := dir + "/reports"
	for _, d := range []string{workspaceDir, reportsDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeSiblingTestFile(t, workspaceDir+"/session.ft", `package workspace

type Session = {
  workDir: String,
}
`)
	writeSiblingTestFile(t, reportsDir+"/summary.ft", `package reports

import "xpkg/workspace"

func buildSummary(ctx workspace.Session): Error {
  return nil
}
`)

	modResult, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	reportsTC := modResult.ForstPackageTypeChecker("reports")
	if reportsTC == nil {
		t.Fatal("missing reports typechecker")
	}
	merged, _, err := forstpkg.ParseAndMergePackage(nil, []string{reportsDir + "/summary.ft"})
	if err != nil {
		t.Fatalf("parse reports: %v", err)
	}
	log := ast.SetupTestLogger(nil)
	log.SetOutput(bytes.NewBuffer(nil))
	tr := New(reportsTC, log, true)
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
	if !strings.Contains(out, "func buildSummary(ctx workspace.Session)") {
		t.Fatalf("expected workspace.Session param, got:\n%s", out)
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
