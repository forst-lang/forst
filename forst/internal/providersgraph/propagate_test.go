package providersgraph

import (
	"testing"

	"forst/internal/ast"
)

func TestPropagateModuleFixedPoint_emptyEdgesNoOp(t *testing.T) {
	t.Parallel()
	perPkg := map[string]map[ast.Identifier][]Slot{
		"alpha": {"F": {{RootIdent: "Logger", Key: "Logger"}}},
	}
	PropagateModuleFixedPoint(perPkg, nil, nil)
	if len(perPkg["alpha"]["F"]) != 1 {
		t.Fatalf("unexpected mutation: %v", perPkg)
	}
}

func TestPropagateModuleFixedPoint_nilSatisfiesUsesDefault(t *testing.T) {
	t.Parallel()
	perPkg := map[string]map[ast.Identifier][]Slot{
		"alpha": {"Callee": {{RootIdent: "Logger", Key: "Logger"}}},
		"beta":  {"Caller": nil},
	}
	PropagateModuleFixedPoint(perPkg, []ModuleCallEdge{{
		CallerPkg: "beta",
		CallerFn:  "Caller",
		TargetPkg: "alpha",
		TargetFn:  "Callee",
	}}, nil)
	if len(perPkg["beta"]["Caller"]) != 1 {
		t.Fatalf("got %v", perPkg["beta"]["Caller"])
	}
}

func TestPropagateModuleFixedPoint_skipsCalleeWithNoSlots(t *testing.T) {
	t.Parallel()
	perPkg := map[string]map[ast.Identifier][]Slot{
		"alpha": {"Callee": nil},
		"beta":  {"Caller": nil},
	}
	PropagateModuleFixedPoint(perPkg, []ModuleCallEdge{{
		CallerPkg: "beta",
		CallerFn:  "Caller",
		TargetPkg: "alpha",
		TargetFn:  "Callee",
	}}, ProviderScopeKeyPresent)
	if len(perPkg["beta"]["Caller"]) != 0 {
		t.Fatalf("expected no propagation from empty callee slots, got %v", perPkg["beta"]["Caller"])
	}
}

func TestPropagateModuleFixedPoint_createsCallerPackageMap(t *testing.T) {
	t.Parallel()
	perPkg := map[string]map[ast.Identifier][]Slot{
		"alpha": {
			"ExpireToken": {{RootIdent: "Logger", Key: "Logger", SourcePkg: "alpha"}},
		},
	}
	PropagateModuleFixedPoint(perPkg, []ModuleCallEdge{{
		CallerPkg: "beta",
		CallerFn:  "Handle",
		TargetPkg: "alpha",
		TargetFn:  "ExpireToken",
	}}, ProviderScopeKeyPresent)
	if len(perPkg["beta"]["Handle"]) != 1 {
		t.Fatalf("got %v", perPkg["beta"]["Handle"])
	}
}

func TestPropagateModuleFixedPoint_multiHopPropagation(t *testing.T) {
	t.Parallel()
	perPkg := map[string]map[ast.Identifier][]Slot{
		"alpha": {"Leaf": {{RootIdent: "Clock", Key: "Clock"}}},
		"beta":  {"Mid": nil},
		"gamma": {"Root": nil},
	}
	edges := []ModuleCallEdge{
		{CallerPkg: "beta", CallerFn: "Mid", TargetPkg: "alpha", TargetFn: "Leaf"},
		{CallerPkg: "gamma", CallerFn: "Root", TargetPkg: "beta", TargetFn: "Mid"},
	}
	PropagateModuleFixedPoint(perPkg, edges, ProviderScopeKeyPresent)
	if len(perPkg["gamma"]["Root"]) != 1 || perPkg["gamma"]["Root"][0].RootIdent != "Clock" {
		t.Fatalf("got %v", perPkg["gamma"]["Root"])
	}
}

func TestPropagateIntraPackageFixedPoint_nilSlotsMap(t *testing.T) {
	t.Parallel()
	direct := map[ast.Identifier]map[string]Slot{
		"f": {"Logger": {RootIdent: "Logger", Key: "Logger"}},
	}
	PropagateIntraPackageFixedPoint(nil, direct, nil, ProviderScopeKeyPresent)
}

func TestPropagateIntraPackageFixedPoint_customSatisfiesSkips(t *testing.T) {
	t.Parallel()
	slots := make(map[ast.Identifier][]Slot)
	direct := map[ast.Identifier]map[string]Slot{
		"callee": { "Logger": {RootIdent: "Logger", Key: "Logger"} },
	}
	edges := []IntraPackageEdge{{
		Caller:        "caller",
		Callee:        "callee",
		ProviderScope: ProviderScopeSnapshot{"Logger": {Ident: "Logger"}},
	}}
	satisfies := func(slot Slot, scope map[string]ast.TypeNode) bool { return true }
	PropagateIntraPackageFixedPoint(slots, direct, edges, satisfies)
	if len(slots["caller"]) != 0 {
		t.Fatalf("expected satisfied scope to skip propagation, got %v", slots["caller"])
	}
}
