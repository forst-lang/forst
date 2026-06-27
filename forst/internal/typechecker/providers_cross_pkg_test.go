package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/providersgraph"
)

func TestPropagateModuleProvidersFixedPoint_crossPackageCall(t *testing.T) {
	perPkg := map[string]map[ast.Identifier][]ProviderSlot{
		"alpha": {
			"ExpireToken": {{RootIdent: "Logger", Key: "Logger", ContractType: ast.TypeNode{Ident: "Logger"}}},
		},
		"beta": {
			"Handle": nil,
		},
	}
	calls := []ModuleCrossCall{{
		CallerPkg: "beta",
		CallerFn:  "Handle",
		TargetPkg: "alpha",
		TargetFn:  "ExpireToken",
	}}
	PropagateModuleProvidersFixedPoint(perPkg, calls, providersgraph.ProviderScopeKeyPresent)
	slots := perPkg["beta"]["Handle"]
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("Handle providers = %v", slots)
	}
}

func TestPropagateModuleProvidersFixedPoint_ambientSatisfiesSkipsSlot(t *testing.T) {
	perPkg := map[string]map[ast.Identifier][]ProviderSlot{
		"alpha": {
			"ExpireToken": {{RootIdent: "Logger", Key: "Logger"}},
		},
		"beta": {
			"Handle": nil,
		},
	}
	calls := []ModuleCrossCall{{
		CallerPkg:   "beta",
		CallerFn:    "Handle",
		TargetPkg:   "alpha",
		TargetFn:    "ExpireToken",
		ProviderScope:     map[string]ast.TypeNode{"Logger": {Ident: "Logger"}},
	}}
	PropagateModuleProvidersFixedPoint(perPkg, calls, providersgraph.ProviderScopeKeyPresent)
	if len(perPkg["beta"]["Handle"]) != 0 {
		t.Fatalf("expected scope to satisfy Logger, got %v", perPkg["beta"]["Handle"])
	}
}

func TestBuildModuleCrossCalls_resolvesForstImportPath(t *testing.T) {
	tc := New(nil, false)
	tc.providers = newProvidersEngine()
	tc.importPathByLocal = map[string]string{"alpha": "testmod/alpha"}
	tc.providers.CallEdges = []providersgraph.CallEdge{{
		CallerFn:    "Handle",
		CalleeFn:    "ExpireToken",
		ImportLocal: "alpha",
	}}
	importMap := map[string]string{"testmod/alpha": "alpha"}
	calls := BuildModuleCrossCalls("beta", tc, importMap)
	if len(calls) != 1 || calls[0].TargetPkg != "alpha" || calls[0].TargetFn != "ExpireToken" {
		t.Fatalf("calls = %+v", calls)
	}
}
