package providersgraph

import (
	"testing"

	"forst/internal/ast"
)

func TestCallEdge_IsCrossPackage(t *testing.T) {
	intra := CallEdge{CalleePkg: ""}
	if intra.IsCrossPackage() {
		t.Fatal("intra-package edge should not be cross-package")
	}
	cross := CallEdge{CalleePkg: "auth"}
	if !cross.IsCrossPackage() {
		t.Fatal("expected cross-package edge")
	}
}

func TestCallEdge_ToModuleCallEdge(t *testing.T) {
	scope := ProviderScopeSnapshot{"Logger": {Ident: "Logger"}}
	e := CallEdge{
		CallerPkg:   "api",
		CallerFn:    "HandleRequest",
		CalleePkg:   "auth",
		CalleeFn:    "LogEvent",
		Scope:       scope,
	}
	got := e.ToModuleCallEdge()
	if got.CallerPkg != "api" || got.CallerFn != "HandleRequest" {
		t.Fatalf("caller fields: %+v", got)
	}
	if got.TargetPkg != "auth" || got.TargetFn != "LogEvent" {
		t.Fatalf("target fields: %+v", got)
	}
	if got.ProviderScope["Logger"].Ident != "Logger" {
		t.Fatalf("scope: %+v", got.ProviderScope)
	}
}

func TestIntraEdgesFromCallEdges_filtersCrossPackage(t *testing.T) {
	edges := []CallEdge{
		{CallerFn: "a", CalleeFn: "b"},
		{CallerFn: "c", CalleeFn: "d", CalleePkg: "other"},
	}
	got := IntraEdgesFromCallEdges(edges)
	if len(got) != 1 {
		t.Fatalf("expected one intra edge, got %d", len(got))
	}
	if got[0].Caller != "a" || got[0].Callee != "b" {
		t.Fatalf("intra edge: %+v", got[0])
	}
}

func TestModuleEdgesFromCallEdges_filtersIntraPackage(t *testing.T) {
	scope := ProviderScopeSnapshot{"K": {Ident: "K"}}
	edges := []CallEdge{
		{CallerFn: "a", CalleeFn: "b"},
		{
			CallerPkg:   "api",
			CallerFn:    "HandleRequest",
			CalleePkg:   "auth",
			CalleeFn:    "LogEvent",
			Scope:       scope,
		},
	}
	got := ModuleEdgesFromCallEdges(edges)
	if len(got) != 1 {
		t.Fatalf("expected one module edge, got %d", len(got))
	}
	if got[0].TargetPkg != "auth" || got[0].TargetFn != "LogEvent" {
		t.Fatalf("module edge: %+v", got[0])
	}
}

func TestModuleEdgesFromCallEdges_emptyCalleePkgSkipped(t *testing.T) {
	got := ModuleEdgesFromCallEdges([]CallEdge{{CallerFn: ast.Identifier("x"), CalleeFn: "y"}})
	if len(got) != 0 {
		t.Fatalf("expected no module edges, got %v", got)
	}
}
