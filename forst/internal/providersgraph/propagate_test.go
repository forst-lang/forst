package providersgraph

import (
	"testing"

	"forst/internal/ast"
)

func TestPropagateModuleFixedPoint_emptyEdgesNoOp(t *testing.T) {
	t.Parallel()
	perPkg := map[string]map[ast.Identifier][]Slot{
		"auth": {"F": {{RootIdent: "Logger", Key: "Logger"}}},
	}
	PropagateModuleFixedPoint(perPkg, nil, nil)
	if len(perPkg["auth"]["F"]) != 1 {
		t.Fatalf("unexpected mutation: %v", perPkg)
	}
}

func TestPropagateModuleFixedPoint_nilSatisfiesUsesDefault(t *testing.T) {
	t.Parallel()
	perPkg := map[string]map[ast.Identifier][]Slot{
		"auth": {"Callee": {{RootIdent: "Logger", Key: "Logger"}}},
		"api":  {"Caller": nil},
	}
	PropagateModuleFixedPoint(perPkg, []ModuleCallEdge{{
		CallerPkg: "api",
		CallerFn:  "Caller",
		TargetPkg: "auth",
		TargetFn:  "Callee",
	}}, nil)
	if len(perPkg["api"]["Caller"]) != 1 {
		t.Fatalf("got %v", perPkg["api"]["Caller"])
	}
}

func TestPropagateModuleFixedPoint_skipsCalleeWithNoSlots(t *testing.T) {
	t.Parallel()
	perPkg := map[string]map[ast.Identifier][]Slot{
		"auth": {"Callee": nil},
		"api":  {"Caller": nil},
	}
	PropagateModuleFixedPoint(perPkg, []ModuleCallEdge{{
		CallerPkg: "api",
		CallerFn:  "Caller",
		TargetPkg: "auth",
		TargetFn:  "Callee",
	}}, ProviderScopeKeyPresent)
	if len(perPkg["api"]["Caller"]) != 0 {
		t.Fatalf("expected no propagation from empty callee slots, got %v", perPkg["api"]["Caller"])
	}
}

func TestPropagateModuleFixedPoint_createsCallerPackageMap(t *testing.T) {
	t.Parallel()
	perPkg := map[string]map[ast.Identifier][]Slot{
		"auth": {
			"LogEvent": {{RootIdent: "Logger", Key: "Logger", SourcePkg: "auth"}},
		},
	}
	PropagateModuleFixedPoint(perPkg, []ModuleCallEdge{{
		CallerPkg: "api",
		CallerFn:  "HandleRequest",
		TargetPkg: "auth",
		TargetFn:  "LogEvent",
	}}, ProviderScopeKeyPresent)
	if len(perPkg["api"]["HandleRequest"]) != 1 {
		t.Fatalf("got %v", perPkg["api"]["HandleRequest"])
	}
}

func TestPropagateModuleFixedPoint_multiHopPropagation(t *testing.T) {
	t.Parallel()
	perPkg := map[string]map[ast.Identifier][]Slot{
		"auth":    {"Leaf": {{RootIdent: "Clock", Key: "Clock"}}},
		"api":     {"Mid": nil},
		"gateway": {"Root": nil},
	}
	edges := []ModuleCallEdge{
		{CallerPkg: "api", CallerFn: "Mid", TargetPkg: "auth", TargetFn: "Leaf"},
		{CallerPkg: "gateway", CallerFn: "Root", TargetPkg: "api", TargetFn: "Mid"},
	}
	PropagateModuleFixedPoint(perPkg, edges, ProviderScopeKeyPresent)
	if len(perPkg["gateway"]["Root"]) != 1 || perPkg["gateway"]["Root"][0].RootIdent != "Clock" {
		t.Fatalf("got %v", perPkg["gateway"]["Root"])
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
	satisfies := func(_ Slot, _ map[string]ast.TypeNode) bool { return true }
	PropagateIntraPackageFixedPoint(slots, direct, edges, satisfies)
	if len(slots["caller"]) != 0 {
		t.Fatalf("expected satisfied scope to skip propagation, got %v", slots["caller"])
	}
}
