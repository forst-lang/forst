package forstcheck_test

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstcheck"
	"forst/internal/goload"
	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/testmod"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestTypecheckFile_tempModule(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "main.ft")
	const src = `package main

func main() {
	println("ok")
}
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(nil)
	nodes := parseTestSource(t, path, src)
	tc, _, err := forstcheck.TypecheckFile(log, forstcheck.TypecheckFileOpts{
		FilePath: path,
		Nodes:    nodes,
	})
	if err != nil {
		t.Fatalf("TypecheckFile: %v", err)
	}
	if tc == nil {
		t.Fatal("expected typechecker")
	}
}

func TestModuleRootForSingleFile_insideCompilerModule(t *testing.T) {
	t.Parallel()
	modRoot := goload.ForstCompilerModuleRoot()
	if modRoot == "" {
		t.Skip("not inside forst compiler module")
	}
	filePath := filepath.Join(modRoot, "cmd", "forst", "probe.ft")
	want := filepath.Join(modRoot, "cmd", "forst")
	if got := forstcheck.ModuleRootForSingleFile(filePath); got != want {
		t.Fatalf("ModuleRootForSingleFile = %q, want %q", got, want)
	}
}

func TestModuleRootForSingleFile_outsideCompilerModule(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("probe_mod")), 0o644); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(dir, "pkg", "main.ft")
	if got := forstcheck.ModuleRootForSingleFile(filePath); got != dir {
		t.Fatalf("ModuleRootForSingleFile = %q, want %q", got, dir)
	}
}

func TestTypecheckFile_multiPackageUsesPerPackageChecker(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("rebind_demo")), 0o644); err != nil {
		t.Fatal(err)
	}
	catalogDir := filepath.Join(dir, "catalog")
	ordersDir := filepath.Join(dir, "orders")
	for _, d := range []string{catalogDir, ordersDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(catalogDir, "types.ft"), []byte(`package catalog

type Config = {
	Version: String,
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	ordersPath := filepath.Join(ordersDir, "use.ft")
	const ordersSrc = `package orders

import "rebind_demo/catalog"

func Use(cfg: catalog.Config) {
	println(cfg.Version)
}
`
	if err := os.WriteFile(ordersPath, []byte(ordersSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(nil)
	nodes := parseTestSource(t, ordersPath, ordersSrc)
	tc, modResult, err := forstcheck.TypecheckFile(log, forstcheck.TypecheckFileOpts{
		FilePath: ordersPath,
		Nodes:    nodes,
	})
	if err != nil {
		t.Fatalf("TypecheckFile: %v", err)
	}
	if modResult == nil || modResult.ForstPackageTypeChecker("orders") == nil {
		t.Fatal("expected module result with orders typechecker")
	}
	if tc == nil {
		t.Fatal("expected typechecker")
	}
}

func TestRebindScopes_preservesFunctionProviders(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(nil)
	const src = `package main

type Logger = { info(msg String) }

func run() {
	use logger: Logger
	logger.info("ok")
}
`
	path := "main.ft"
	nodes := parseTestSource(t, path, src)
	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("initial typecheck: %v", err)
	}
	if len(tc.FunctionProviders) == 0 {
		t.Fatal("expected function providers from initial check")
	}
	before := len(tc.FunctionProviders)
	if err := forstcheck.RebindScopes(tc, nodes); err != nil {
		t.Fatalf("RebindScopes: %v", err)
	}
	if len(tc.FunctionProviders) != before {
		t.Fatalf("providers count changed: before=%d after=%d", before, len(tc.FunctionProviders))
	}
}

func parseTestSource(t *testing.T, path, src string) []ast.Node {
	t.Helper()
	lex := lexer.New([]byte(src), path, logrus.New())
	tokens := lex.Lex()
	psr := parser.New(tokens, path, logrus.New())
	nodes, err := psr.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return nodes
}
