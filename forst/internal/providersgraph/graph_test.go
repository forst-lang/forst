package providersgraph

import (
	"testing"

	"forst/internal/ast"
)

func TestPropagateModuleFixedPoint_crossPackageCall(t *testing.T) {
	perPkg := map[string]map[ast.Identifier][]Slot{
		"alpha": {
			"ExpireToken": {{RootIdent: "Logger", Key: "Logger", ContractType: ast.TypeNode{Ident: "Logger"}}},
		},
		"beta": {
			"Handle": nil,
		},
	}
	edges := []ModuleCallEdge{{
		CallerPkg: "beta",
		CallerFn:  "Handle",
		TargetPkg: "alpha",
		TargetFn:  "ExpireToken",
	}}
	PropagateModuleFixedPoint(perPkg, edges, ProviderScopeKeyPresent)
	slots := perPkg["beta"]["Handle"]
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("Handle providers = %v", slots)
	}
}

func TestPropagateModuleFixedPoint_ambientSatisfiesSkipsSlot(t *testing.T) {
	perPkg := map[string]map[ast.Identifier][]Slot{
		"alpha": {
			"ExpireToken": {{RootIdent: "Logger", Key: "Logger"}},
		},
		"beta": {
			"Handle": nil,
		},
	}
	edges := []ModuleCallEdge{{
		CallerPkg:   "beta",
		CallerFn:    "Handle",
		TargetPkg:   "alpha",
		TargetFn:    "ExpireToken",
		ProviderScope:     ProviderScopeSnapshot{"Logger": {Ident: "Logger"}},
	}}
	PropagateModuleFixedPoint(perPkg, edges, ProviderScopeKeyPresent)
	if len(perPkg["beta"]["Handle"]) != 0 {
		t.Fatalf("expected scope to satisfy Logger, got %v", perPkg["beta"]["Handle"])
	}
}

func TestPropagateIntraPackageFixedPoint_propagatesCalleeToCaller(t *testing.T) {
	slots := make(map[ast.Identifier][]Slot)
	direct := map[ast.Identifier]map[string]Slot{
		"callee": {
			"Logger": {RootIdent: "Logger", Key: "Logger"},
		},
	}
	edges := []IntraPackageEdge{{
		Caller:  "caller",
		Callee:  "callee",
		ProviderScope: nil,
	}}
	PropagateIntraPackageFixedPoint(slots, direct, edges, ProviderScopeKeyPresent)
	got := slots["caller"]
	if len(got) != 1 || got[0].RootIdent != "Logger" {
		t.Fatalf("caller providers = %v", got)
	}
}

func TestGraphInvalidate_transitiveCallers(t *testing.T) {
	g := New()
	g.AddIntraCall("a", "b", nil)
	g.AddIntraCall("b", "c", nil)
	invalid := g.Invalidate("c")
	if len(invalid) != 3 {
		t.Fatalf("Invalidate(c) = %v, want a,b,c", invalid)
	}
}

func TestGraph_SetDirectAndSlots(t *testing.T) {
	g := New()
	g.SetDirect("fn", map[string]Slot{
		"Logger": {RootIdent: "Logger", Key: "Logger"},
	})
	if len(g.direct["fn"]) != 1 {
		t.Fatalf("direct slots: %v", g.direct["fn"])
	}
	g.SetDirect("empty", nil)
	if _, ok := g.direct["empty"]; ok {
		t.Fatal("empty direct map should not be stored")
	}
	g.ComputeIntraFixedPoint(ProviderScopeKeyPresent)
	if len(g.Slots("fn")) != 1 {
		t.Fatalf("Slots(fn) = %v", g.Slots("fn"))
	}
	all := g.AllSlots()
	if len(all["fn"]) != 1 {
		t.Fatalf("AllSlots = %v", all)
	}
}

func TestGraph_AddModuleCallAndIntraCall(t *testing.T) {
	g := New()
	g.AddIntraCall("caller", "callee", ProviderScopeSnapshot{"K": {Ident: "K"}})
	if len(g.intraEdges) != 1 {
		t.Fatalf("intra edges: %v", g.intraEdges)
	}
	g.AddModuleCall(ModuleCallEdge{
		CallerPkg: "beta",
		CallerFn:  "Handle",
		TargetPkg: "alpha",
		TargetFn:  "ExpireToken",
	})
	if len(g.moduleEdges) != 1 {
		t.Fatalf("module edges: %v", g.moduleEdges)
	}
}

func TestModuleGraph_AllPackages(t *testing.T) {
	perPkg := map[string]map[ast.Identifier][]Slot{
		"alpha": {"F": {{RootIdent: "Logger", Key: "Logger"}}},
	}
	mg := NewModuleGraph(perPkg)
	all := mg.AllPackages()
	if len(all["alpha"]["F"]) != 1 {
		t.Fatalf("AllPackages = %v", all)
	}
	all["alpha"]["F"][0].RootIdent = "mutated"
	if mg.PerPackage("alpha")["F"][0].RootIdent != "Logger" {
		t.Fatal("AllPackages should return a deep copy")
	}
}

func TestModuleGraph_emptyPackageClone(t *testing.T) {
	mg := NewModuleGraph(map[string]map[ast.Identifier][]Slot{
		"empty": {},
	})
	all := mg.AllPackages()
	if len(all["empty"]) != 0 {
		t.Fatalf("expected empty fn map, got %v", all["empty"])
	}
}

func TestGraph_Invalidate_skipsDuplicateCallers(t *testing.T) {
	g := New()
	g.AddIntraCall("a", "b", nil)
	g.AddIntraCall("a", "b", nil)
	invalid := g.Invalidate("b")
	if len(invalid) != 2 {
		t.Fatalf("got %v", invalid)
	}
}

func TestModuleGraph_fixedPoint(t *testing.T) {
	perPkg := map[string]map[ast.Identifier][]Slot{
		"alpha": {"ExpireToken": {{RootIdent: "Logger", Key: "Logger"}}},
		"beta":  {"Handle": nil},
	}
	mg := NewModuleGraph(perPkg)
	mg.AddModuleCall(ModuleCallEdge{
		CallerPkg: "beta",
		CallerFn:  "Handle",
		TargetPkg: "alpha",
		TargetFn:  "ExpireToken",
	})
	mg.ComputeFixedPoint(ProviderScopeKeyPresent)
	slots := mg.PerPackage("beta")["Handle"]
	if len(slots) != 1 {
		t.Fatalf("Handle providers = %v", slots)
	}
}
