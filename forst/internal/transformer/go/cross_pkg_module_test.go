package transformergo

import (
	"bytes"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/modulecheck"
	"forst/internal/testmod"
	"forst/internal/typechecker"
)

func writeCrossPkgModule(t *testing.T, dir string) {
	t.Helper()
	testmod.WriteGoMod(t, dir, "xpkg")
	alphaDir := filepath.Join(dir, "alpha")
	betaDir := filepath.Join(dir, "beta")
	for _, d := range []string{alphaDir, betaDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(alphaDir, "log.ft"), `package alpha

type Logger = { Info(msg String) }
type NopLogger = {}

func (NopLogger) Info(msg String) {}

func LogExpiry(id String) {
	use logger: Logger
	logger.Info("expire " + id)
}
`)
	writeFile(t, filepath.Join(betaDir, "handle.ft"), `package beta

import "xpkg/alpha"

type Logger = { Info(msg String) }

func Handle(id String) {
	alpha.LogExpiry(id)
}
`)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func transformBetaPackage(t *testing.T, moduleRoot string) string {
	t.Helper()
	modResult, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: moduleRoot})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	betaTC := modResult.ForstPackageTypeChecker("beta")
	if betaTC == nil {
		t.Fatal("missing beta typechecker")
	}
	betaPath := filepath.Join(moduleRoot, "beta", "handle.ft")
	merged, _, err := forstpkg.ParseAndMergePackage(nil, []string{betaPath})
	if err != nil {
		t.Fatalf("parse beta: %v", err)
	}
	log := ast.SetupTestLogger(nil)
	log.SetOutput(bytes.NewBuffer(nil))
	tr := New(betaTC, log)
	tr.SetModuleResult(modResult)
	goFile, err := tr.TransformForstFileToGo(merged)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), goFile); err != nil {
		t.Fatalf("format: %v", err)
	}
	return buf.String()
}

func TestCrossPkgModule_HandleForwardsProvidersToAlpha(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeCrossPkgModule(t, dir)
	out := transformBetaPackage(t, dir)
	for _, sub := range []string{
		`func Handle(providers`,
		`Providers_`,
		`alpha.LogExpiry(`,
		`alpha.Providers_`,
		`Logger:`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("missing %q in:\n%s", sub, out)
		}
	}
	assertGoParses(t, out)
}

func TestBuildCrossPackageProvidersLiteral_unit(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	tr.currentFnProvidersName = "providers_"
	slots := []typechecker.ProviderSlot{{RootIdent: "Logger", ContractType: ast.TypeNode{Ident: "Logger"}}}
	lit, err := tr.buildCrossPackageProvidersLiteral("alpha", slots)
	if err != nil {
		t.Fatal(err)
	}
	s := goExprString(t, lit)
	if !strings.Contains(s, "alpha.Providers_") || !strings.Contains(s, "providers_.Logger") {
		t.Fatalf("got %s", s)
	}
}

func TestImportLocalForForstPkg_viaModuleResult(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeCrossPkgModule(t, dir)
	modResult, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	betaTC := modResult.ForstPackageTypeChecker("beta")
	if betaTC == nil {
		t.Fatal("missing beta tc")
	}
	got := importLocalForForstPkg(betaTC, modResult, "alpha")
	if got != "alpha" {
		t.Fatalf("import local = %q, want alpha", got)
	}
}

func TestCalleeProviderSlots_crossPackageQualified(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeCrossPkgModule(t, dir)
	modResult, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	betaTC := modResult.ForstPackageTypeChecker("beta")
	log := setupTestLogger(nil)
	tr := New(betaTC, log)
	tr.SetModuleResult(modResult)
	slots := tr.calleeProviderSlots("alpha.LogExpiry")
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("slots = %#v", slots)
	}
}
