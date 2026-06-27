package usablesgraph

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
	PropagateModuleFixedPoint(perPkg, edges, AmbientKeyPresent)
	slots := perPkg["beta"]["Handle"]
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("Handle usables = %v", slots)
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
		Ambient:     AmbientSnapshot{"Logger": {Ident: "Logger"}},
	}}
	PropagateModuleFixedPoint(perPkg, edges, AmbientKeyPresent)
	if len(perPkg["beta"]["Handle"]) != 0 {
		t.Fatalf("expected ambient to satisfy Logger, got %v", perPkg["beta"]["Handle"])
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
		Ambient: nil,
	}}
	PropagateIntraPackageFixedPoint(slots, direct, edges, AmbientKeyPresent)
	got := slots["caller"]
	if len(got) != 1 || got[0].RootIdent != "Logger" {
		t.Fatalf("caller usables = %v", got)
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
	mg.ComputeFixedPoint(AmbientKeyPresent)
	slots := mg.PerPackage("beta")["Handle"]
	if len(slots) != 1 {
		t.Fatalf("Handle usables = %v", slots)
	}
}
