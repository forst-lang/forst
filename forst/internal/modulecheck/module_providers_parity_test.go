package modulecheck_test

import (
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/compiler"
	"forst/internal/discovery"
	"forst/internal/modulecheck"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func crossPkgWantRoots() []string {
	return []string{"Logger"}
}

func TestModuleProviders_crossPkg_modulecheck(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "rfc", "providers", "cross_pkg")
	wantRoots := crossPkgWantRoots()

	modResult, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: root})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	modRoots := providerRoots(modResult.ForstPackageTypeChecker("api").FunctionProviders, "HandleRequest")
	if !sameStringSlice(modRoots, wantRoots) {
		t.Fatalf("modulecheck HandleRequest roots = %v, want %v", modRoots, wantRoots)
	}
	modSlots := modResult.ForstPackageTypeChecker("api").FunctionProviders["HandleRequest"]
	if len(modSlots) != 1 || modSlots[0].Key != "Logger" {
		t.Fatalf("modulecheck HandleRequest slots = %v", modSlots)
	}
}

func TestModuleProviders_crossPkg_discovery(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "rfc", "providers", "cross_pkg")
	wantRoots := crossPkgWantRoots()

	logger := logrus.New()
	logger.SetOutput(nil)
	logger.SetLevel(logrus.PanicLevel)
	crossPkgFiles := []string{
		filepath.Join(root, "auth", "log.ft"),
		filepath.Join(root, "api", "handle.ft"),
		filepath.Join(root, "api", "handle_test.ft"),
	}
	discoverer := discovery.NewDiscoverer(root, logger, discovery.NewStaticFilesConfig(crossPkgFiles))
	functions, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	handle, ok := functions["api"]["HandleRequest"]
	if !ok {
		t.Fatalf("missing api.HandleRequest in discovery: %+v", functions)
	}
	if !sameStringSlice(handle.Providers, wantRoots) {
		t.Fatalf("discovery HandleRequest providers = %v, want %v", handle.Providers, wantRoots)
	}
}

func TestModuleProviders_crossPkg_compileTypecheck(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "rfc", "providers", "cross_pkg")
	wantRoots := crossPkgWantRoots()

	logger := logrus.New()
	logger.SetOutput(nil)
	logger.SetLevel(logrus.PanicLevel)
	entry := filepath.Join(root, "api", "handle.ft")
	c := compiler.New(compiler.Args{
		Command:  "run",
		FilePath: entry,
		LogLevel: "error",
	}, logger)
	tc, modFromCompile, err := c.TypecheckForCompileEntry()
	if err != nil {
		t.Fatalf("compile typecheck: %v", err)
	}
	if modFromCompile == nil {
		t.Fatal("expected module result from compile typecheck")
	}
	compileTC := modFromCompile.ForstPackageTypeChecker("api")
	if compileTC == nil {
		t.Fatal("missing api tc from compile module pass")
	}
	compileRoots := providerRoots(compileTC.FunctionProviders, "HandleRequest")
	if !sameStringSlice(compileRoots, wantRoots) {
		t.Fatalf("compile HandleRequest roots = %v, want %v", compileRoots, wantRoots)
	}
	if tc != compileTC {
		directRoots := providerRoots(tc.FunctionProviders, "HandleRequest")
		if !sameStringSlice(directRoots, wantRoots) {
			t.Fatalf("returned tc HandleRequest roots = %v, want %v", directRoots, wantRoots)
		}
	}
}

func providerRoots(slots map[ast.Identifier][]typechecker.ProviderSlot, fn string) []string {
	return typechecker.ProviderRootIdentsFromSlots(slots[ast.Identifier(fn)])
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
