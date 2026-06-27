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

func TestModuleProviders_parity_discoveryCompileModulecheck_crossPkg(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "rfc", "providers", "cross_pkg")
	wantRoots := []string{"Logger"}

	modResult, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: root})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	modRoots := providerRoots(modResult.ForstPackageTypeChecker("beta").FunctionProviders, "Handle")
	if !sameStringSlice(modRoots, wantRoots) {
		t.Fatalf("modulecheck Handle roots = %v, want %v", modRoots, wantRoots)
	}
	modSlots := modResult.ForstPackageTypeChecker("beta").FunctionProviders["Handle"]
	if len(modSlots) != 1 || modSlots[0].Key != "Logger" {
		t.Fatalf("modulecheck Handle slots = %v", modSlots)
	}

	logger := logrus.New()
	logger.SetOutput(nil)
	logger.SetLevel(logrus.PanicLevel)
	crossPkgFiles := []string{
		filepath.Join(root, "alpha", "log.ft"),
		filepath.Join(root, "beta", "handle.ft"),
		filepath.Join(root, "beta", "handle_test.ft"),
	}
	discoverer := discovery.NewDiscoverer(root, logger, discovery.NewStaticFilesConfig(crossPkgFiles))
	functions, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	handle, ok := functions["beta"]["Handle"]
	if !ok {
		t.Fatalf("missing beta.Handle in discovery: %+v", functions)
	}
	if !sameStringSlice(handle.Providers, wantRoots) {
		t.Fatalf("discovery Handle providers = %v, want %v", handle.Providers, wantRoots)
	}

	entry := filepath.Join(root, "beta", "handle.ft")
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
	compileTC := modFromCompile.ForstPackageTypeChecker("beta")
	if compileTC == nil {
		t.Fatal("missing beta tc from compile module pass")
	}
	compileRoots := providerRoots(compileTC.FunctionProviders, "Handle")
	if !sameStringSlice(compileRoots, wantRoots) {
		t.Fatalf("compile Handle roots = %v, want %v", compileRoots, wantRoots)
	}
	if tc != compileTC {
		// TypecheckForTest may return the same package tc; roots must still match direct lookup.
		directRoots := providerRoots(tc.FunctionProviders, "Handle")
		if !sameStringSlice(directRoots, wantRoots) {
			t.Fatalf("returned tc Handle roots = %v, want %v", directRoots, wantRoots)
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
