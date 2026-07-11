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
	authDir := filepath.Join(dir, "auth")
	apiDir := filepath.Join(dir, "api")
	for _, d := range []string{authDir, apiDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(authDir, "log.ft"), `package auth

type Logger = { Info(msg String) }
type NopLogger = {}

func (NopLogger) Info(msg String) {}

func LogEvent(id String) {
	use logger: Logger
	logger.Info("expire " + id)
}
`)
	writeFile(t, filepath.Join(apiDir, "handle.ft"), `package api

import "xpkg/auth"

type Logger = { Info(msg String) }

func HandleRequest(id String) {
	auth.LogEvent(id)
}
`)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func transformAPIPackage(t *testing.T, moduleRoot string) string {
	t.Helper()
	modResult, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: moduleRoot})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	apiTC := modResult.ForstPackageTypeChecker("api")
	if apiTC == nil {
		t.Fatal("missing api typechecker")
	}
	apiPath := filepath.Join(moduleRoot, "api", "handle.ft")
	merged, _, err := forstpkg.ParseAndMergePackage(nil, []string{apiPath})
	if err != nil {
		t.Fatalf("parse api: %v", err)
	}
	log := ast.SetupTestLogger(nil)
	log.SetOutput(bytes.NewBuffer(nil))
	tr := New(apiTC, log)
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
	out := transformAPIPackage(t, dir)
	for _, sub := range []string{
		`func HandleRequest(providers`,
		`Providers_`,
		`auth.LogEvent(`,
		`auth.Providers_`,
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
	lit, err := tr.buildCrossPackageProvidersLiteral("auth", slots)
	if err != nil {
		t.Fatal(err)
	}
	s := goExprString(t, lit)
	if !strings.Contains(s, "auth.Providers_") || !strings.Contains(s, "providers_.Logger") {
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
	apiTC := modResult.ForstPackageTypeChecker("api")
	if apiTC == nil {
		t.Fatal("missing api tc")
	}
	got := importLocalForForstPkg(apiTC, modResult, "auth")
	if got != "auth" {
		t.Fatalf("import local = %q, want auth", got)
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
	apiTC := modResult.ForstPackageTypeChecker("api")
	log := setupTestLogger(nil)
	tr := New(apiTC, log)
	tr.SetModuleResult(modResult)
	slots := tr.calleeProviderSlots("auth.LogEvent")
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("slots = %#v", slots)
	}
}
