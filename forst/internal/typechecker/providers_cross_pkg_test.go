package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/providersgraph"
)

func TestPropagateModuleProvidersFixedPoint_crossPackageCall(t *testing.T) {
	perPkg := map[string]map[ast.Identifier][]ProviderSlot{
		"auth": {
			"LogEvent": {{RootIdent: "Logger", Key: "Logger", ContractType: ast.TypeNode{Ident: "Logger"}}},
		},
		"api": {
			"HandleRequest": nil,
		},
	}
	calls := []ModuleCrossCall{{
		CallerPkg: "api",
		CallerFn:  "HandleRequest",
		TargetPkg: "auth",
		TargetFn:  "LogEvent",
	}}
	PropagateModuleProvidersFixedPoint(perPkg, calls, providersgraph.ProviderScopeKeyPresent)
	slots := perPkg["api"]["HandleRequest"]
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("HandleRequest providers = %v", slots)
	}
}

func TestPropagateModuleProvidersFixedPoint_ambientSatisfiesSkipsSlot(t *testing.T) {
	perPkg := map[string]map[ast.Identifier][]ProviderSlot{
		"auth": {
			"LogEvent": {{RootIdent: "Logger", Key: "Logger"}},
		},
		"api": {
			"HandleRequest": nil,
		},
	}
	calls := []ModuleCrossCall{{
		CallerPkg:     "api",
		CallerFn:      "HandleRequest",
		TargetPkg:     "auth",
		TargetFn:      "LogEvent",
		ProviderScope: map[string]ast.TypeNode{"Logger": {Ident: "Logger"}},
	}}
	PropagateModuleProvidersFixedPoint(perPkg, calls, providersgraph.ProviderScopeKeyPresent)
	if len(perPkg["api"]["HandleRequest"]) != 0 {
		t.Fatalf("expected scope to satisfy Logger, got %v", perPkg["api"]["HandleRequest"])
	}
}

func TestBuildModuleCrossCalls_resolvesForstImportPath(t *testing.T) {
	tc := New(nil, false)
	tc.providers = newProvidersEngine()
	tc.importPathByLocal = map[string]string{"auth": "testmod/auth"}
	tc.providers.CallEdges = []providersgraph.CallEdge{{
		CallerFn:    "HandleRequest",
		CalleeFn:    "LogEvent",
		ImportLocal: "auth",
	}}
	importMap := map[string]string{"testmod/auth": "auth"}
	calls := BuildModuleCrossCalls("api", tc, importMap)
	if len(calls) != 1 || calls[0].TargetPkg != "auth" || calls[0].TargetFn != "LogEvent" {
		t.Fatalf("calls = %+v", calls)
	}
}
