package providersgraph

import (
	"testing"

	"forst/internal/ast"
)

func TestPropagateModuleFixedPoint_cyclicCrossPackage(t *testing.T) {
	perPkg := map[string]map[ast.Identifier][]Slot{
		"alpha": {
			"Log": {{RootIdent: "Logger", Key: "Logger", ContractType: ast.TypeNode{Ident: "Logger"}}},
		},
		"beta": {},
	}
	edges := []ModuleCallEdge{
		{CallerPkg: "beta", CallerFn: "Handle", TargetPkg: "alpha", TargetFn: "Log"},
		{CallerPkg: "alpha", CallerFn: "Log", TargetPkg: "beta", TargetFn: "Handle"},
	}
	PropagateModuleFixedPoint(perPkg, edges, ProviderScopeKeyPresent)
	if len(perPkg["beta"]["Handle"]) == 0 {
		t.Fatal("beta.Handle should inherit Logger from cyclic cross-package graph")
	}
	if len(perPkg["alpha"]["Log"]) == 0 {
		t.Fatal("alpha.Log should retain Logger")
	}
}
