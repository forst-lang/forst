package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestPropagateModuleUsablesFixedPoint_crossPackageCall(t *testing.T) {
	perPkg := map[string]map[ast.Identifier][]UsableSlot{
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
	PropagateModuleUsablesFixedPoint(perPkg, calls)
	slots := perPkg["beta"]["Handle"]
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("Handle usables = %v", slots)
	}
}

func TestPropagateModuleUsablesFixedPoint_ambientSatisfiesSkipsSlot(t *testing.T) {
	perPkg := map[string]map[ast.Identifier][]UsableSlot{
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
		Ambient:     map[string]ast.TypeNode{"Logger": {Ident: "Logger"}},
	}}
	PropagateModuleUsablesFixedPoint(perPkg, calls)
	if len(perPkg["beta"]["Handle"]) != 0 {
		t.Fatalf("expected ambient to satisfy Logger, got %v", perPkg["beta"]["Handle"])
	}
}

func TestBuildModuleCrossCalls_resolvesForstImportPath(t *testing.T) {
	tc := New(nil, false)
	tc.importPathByLocal = map[string]string{"alpha": "testmod/alpha"}
	tc.crossPackageCallSites = []crossPackageCallRecord{{
		CallerFn:    "Handle",
		ImportLocal: "alpha",
		CalleeFn:    "ExpireToken",
	}}
	importMap := map[string]string{"testmod/alpha": "alpha"}
	calls := BuildModuleCrossCalls("beta", tc, importMap)
	if len(calls) != 1 || calls[0].TargetPkg != "alpha" || calls[0].TargetFn != "ExpireToken" {
		t.Fatalf("calls = %+v", calls)
	}
}
