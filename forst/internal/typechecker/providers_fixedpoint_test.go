package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/providersgraph"

	"github.com/sirupsen/logrus"
)

func TestBuildGraph_mirrorsDirectAndIntraEdges(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.providers = newProvidersEngine()
	tc.providers.Direct = map[ast.Identifier]map[string]ProviderSlot{
		"inner": {"Logger": {RootIdent: "Logger", Key: "Logger"}},
	}
	tc.providers.CallEdges = []providersgraph.CallEdge{{
		CallerFn: "outer",
		CalleeFn: "inner",
		Scope:    providersgraph.ProviderScopeSnapshot{},
	}}

	g := tc.BuildGraph()
	g.ComputeIntraFixedPoint(tc.scopeSatisfiesSlot)
	slots := g.Slots("outer")
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("outer slots = %v", slots)
	}
	if len(g.Slots("inner")) != 1 {
		t.Fatalf("inner slots = %v", g.Slots("inner"))
	}
}

func TestValidateCallSite_callerForwardsSkipsError(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.FunctionProviders = map[ast.Identifier][]ProviderSlot{
		"outer": {
			{RootIdent: "Logger", Key: "Logger"},
			{RootIdent: "Clock", Key: "Clock"},
		},
		"inner": {{RootIdent: "Logger", Key: "Logger"}},
	}
	edge := providersgraph.CallEdge{
		CallerFn: "outer",
		CalleeFn: "inner",
		Scope:    providersgraph.ProviderScopeSnapshot{},
	}
	if err := tc.validateCallSite("outer", edge); err != nil {
		t.Fatalf("validateCallSite: %v", err)
	}
}

func TestValidateCallSite_unsatisfiedEmitsDiagnostic(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.Functions = map[ast.Identifier]FunctionSignature{
		"outer": {Ident: ast.Ident{ID: "outer", Span: ast.SourceSpan{StartLine: 1}}},
		"inner": {Ident: ast.Ident{ID: "inner", Span: ast.SourceSpan{StartLine: 2}}},
	}
	tc.FunctionProviders = map[ast.Identifier][]ProviderSlot{
		"outer": {},
		"inner": {{RootIdent: "Logger", Key: "Logger", ContractType: ast.TypeNode{Ident: "Logger"}}},
	}
	edge := providersgraph.CallEdge{
		CallerFn: "outer",
		CalleeFn: "inner",
		Scope:    providersgraph.ProviderScopeSnapshot{},
		Span:     ast.SourceSpan{StartLine: 3},
	}
	err := tc.validateCallSite("outer", edge)
	if err == nil {
		t.Fatal("expected unsatisfied providers error")
	}
	diag, ok := err.(*Diagnostic)
	if !ok || diag.Code != "providers-unsatisfied" {
		t.Fatalf("got %T %v", err, err)
	}
	if !strings.Contains(diag.Msg, "Logger") {
		t.Fatalf("msg = %q", diag.Msg)
	}
}

func TestValidateIntraCallSites_callerForwardsSkipsValidation(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.providers = newProvidersEngine()
	tc.providers.CallEdges = []providersgraph.CallEdge{{
		CallerFn: "outer",
		CalleeFn: "inner",
		Scope:    providersgraph.ProviderScopeSnapshot{},
	}}
	tc.FunctionProviders = map[ast.Identifier][]ProviderSlot{
		"outer": {{RootIdent: "Logger", Key: "Logger"}},
		"inner": {{RootIdent: "Logger", Key: "Logger"}},
	}
	if err := tc.validateIntraCallSites(); err != nil {
		t.Fatalf("validateIntraCallSites: %v", err)
	}
}
