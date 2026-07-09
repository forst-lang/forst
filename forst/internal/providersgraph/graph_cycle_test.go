package providersgraph

import (
	"testing"

	"forst/internal/ast"
)

func TestPropagateModuleFixedPoint_cyclicCrossPackage(t *testing.T) {
	perPkg := map[string]map[ast.Identifier][]Slot{
		"auth": {
			"LogEvent": {{RootIdent: "Logger", Key: "Logger", ContractType: ast.TypeNode{Ident: "Logger"}}},
		},
		"api": {},
	}
	edges := []ModuleCallEdge{
		{CallerPkg: "api", CallerFn: "HandleRequest", TargetPkg: "auth", TargetFn: "LogEvent"},
		{CallerPkg: "auth", CallerFn: "LogEvent", TargetPkg: "api", TargetFn: "HandleRequest"},
	}
	PropagateModuleFixedPoint(perPkg, edges, ProviderScopeKeyPresent)
	if len(perPkg["api"]["HandleRequest"]) == 0 {
		t.Fatal("api.HandleRequest should inherit Logger from cyclic cross-package graph")
	}
	if len(perPkg["auth"]["LogEvent"]) == 0 {
		t.Fatal("auth.LogEvent should retain Logger")
	}
}
