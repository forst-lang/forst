package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/providersgraph"

	"github.com/sirupsen/logrus"
)

func TestValidateSidecarExportable_rejectsProvidersOnExportedFn(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.FunctionProviders = map[ast.Identifier][]ProviderSlot{
		"Handle": {{RootIdent: "Logger", Key: "Logger"}},
	}
	nodes := []ast.Node{
		ast.PackageNode{Ident: ast.Ident{ID: "main"}},
		ast.FunctionNode{Ident: ast.Ident{ID: "Handle", Span: ast.SourceSpan{StartLine: 1}}},
	}
	err := tc.validateSidecarExportable(nodes)
	if err == nil {
		t.Fatal("expected sidecar export error")
	}
	diag, ok := err.(*Diagnostic)
	if !ok || diag.Code != "providers-sidecar-export" {
		t.Fatalf("got %T %v", err, err)
	}
	if !strings.Contains(diag.Msg, "Handle") || !strings.Contains(diag.Msg, "Logger") {
		t.Fatalf("msg = %q", diag.Msg)
	}
}

func TestValidateSidecarExportable_nonMainPackageSkipped(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.FunctionProviders = map[ast.Identifier][]ProviderSlot{
		"Handle": {{RootIdent: "Logger", Key: "Logger"}},
	}
	nodes := []ast.Node{
		ast.PackageNode{Ident: ast.Ident{ID: "lib"}},
		ast.FunctionNode{Ident: ast.Ident{ID: "Handle"}},
	}
	if err := tc.validateSidecarExportable(nodes); err != nil {
		t.Fatal(err)
	}
}

func TestValidateWiringRootFn_unsatisfiedProviders(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.Functions = map[ast.Identifier]FunctionSignature{
		"TestX": {Ident: ast.Ident{ID: "TestX", Span: ast.SourceSpan{StartLine: 2}}},
	}
	tc.FunctionProviders = map[ast.Identifier][]ProviderSlot{
		"TestX": {{RootIdent: "Logger", Key: "Logger"}},
	}
	err := tc.validateWiringRootFn("TestX")
	if err == nil {
		t.Fatal("expected wiring root error")
	}
	diag, ok := err.(*Diagnostic)
	if !ok || diag.Code != "providers-unsatisfied" {
		t.Fatalf("got %T %v", err, err)
	}
	if !strings.Contains(diag.Msg, "Logger") {
		t.Fatalf("msg = %q", diag.Msg)
	}
}

func TestValidateWiringRootFn_noProvidersOk(t *testing.T) {
	tc := New(logrus.New(), false)
	if err := tc.validateWiringRootFn("TestX"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateModuleProviders_crossPackageUnsatisfied(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.providers = newProvidersEngine()
	tc.importPathByLocal = map[string]string{"auth": "testmod/auth"}
	tc.providers.CallEdges = []providersgraph.CallEdge{{
		CallerFn:    "HandleRequest",
		CalleeFn:    "LogEvent",
		ImportLocal: "auth",
		Scope:       providersgraph.ProviderScopeSnapshot{},
		Span:        ast.SourceSpan{StartLine: 5},
	}}
	perPkg := map[string]map[ast.Identifier][]ProviderSlot{
		"auth": {"LogEvent": {{RootIdent: "Logger", Key: "Logger"}}},
	}
	importMap := map[string]string{"testmod/auth": "auth"}
	err := ValidateModuleProviders("api", tc, importMap, perPkg)
	if err == nil {
		t.Fatal("expected cross-package unsatisfied error")
	}
	diag, ok := err.(*Diagnostic)
	if !ok || diag.Code != "providers-unsatisfied" {
		t.Fatalf("got %T %v", err, err)
	}
}

func TestScopeSatisfiesAllSlots(t *testing.T) {
	tc := New(logrus.New(), false)
	scope := map[string]ast.TypeNode{"Logger": {Ident: "Logger"}}
	slots := []ProviderSlot{{RootIdent: "Logger", Key: "Logger", ContractType: ast.TypeNode{Ident: "Logger"}}}
	if !scopeSatisfiesAllSlots(tc, scope, slots) {
		t.Fatal("expected scope to satisfy slot")
	}
	if scopeSatisfiesAllSlots(tc, map[string]ast.TypeNode{}, slots) {
		t.Fatal("expected empty scope to fail")
	}
}

func TestCallerForwardsSlots(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.FunctionProviders = map[ast.Identifier][]ProviderSlot{
		"caller": {{RootIdent: "Logger", Key: "Logger"}, {RootIdent: "Clock", Key: "Clock"}},
	}
	callee := []ProviderSlot{{RootIdent: "Logger", Key: "Logger"}}
	if !callerForwardsSlots(tc, "caller", callee) {
		t.Fatal("caller should forward Logger")
	}
	if callerForwardsSlots(tc, "caller", []ProviderSlot{{RootIdent: "Missing", Key: "Missing"}}) {
		t.Fatal("caller should not forward missing slot")
	}
}

func TestValidateModuleProviders_nilTcReturnsNil(t *testing.T) {
	if err := ValidateModuleProviders("api", nil, nil, nil); err != nil {
		t.Fatalf("ValidateModuleProviders(nil tc): %v", err)
	}
}

func TestValidateModuleProviders_satisfiedCrossPackageOk(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.providers = newProvidersEngine()
	tc.importPathByLocal = map[string]string{"auth": "testmod/auth"}
	tc.providers.CallEdges = []providersgraph.CallEdge{{
		CallerFn:    "HandleRequest",
		CalleeFn:    "LogEvent",
		ImportLocal: "auth",
		Scope:       map[string]ast.TypeNode{"Logger": {Ident: "Logger"}},
	}}
	perPkg := map[string]map[ast.Identifier][]ProviderSlot{
		"auth": {"LogEvent": {{RootIdent: "Logger", Key: "Logger", ContractType: ast.TypeNode{Ident: "Logger"}}},
		},
	}
	importMap := map[string]string{"testmod/auth": "auth"}
	if err := ValidateModuleProviders("api", tc, importMap, perPkg); err != nil {
		t.Fatalf("ValidateModuleProviders: %v", err)
	}
}

func TestValidateModuleProviders_callerForwardsCrossPackage(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.providers = newProvidersEngine()
	tc.importPathByLocal = map[string]string{"auth": "testmod/auth"}
	tc.providers.CallEdges = []providersgraph.CallEdge{{
		CallerFn:    "HandleRequest",
		CalleeFn:    "LogEvent",
		ImportLocal: "auth",
		Scope:       providersgraph.ProviderScopeSnapshot{},
	}}
	tc.FunctionProviders = map[ast.Identifier][]ProviderSlot{
		"HandleRequest": {
			{RootIdent: "Logger", Key: "Logger"},
			{RootIdent: "Clock", Key: "Clock"},
		},
	}
	perPkg := map[string]map[ast.Identifier][]ProviderSlot{
		"auth": {"LogEvent": {{RootIdent: "Logger", Key: "Logger", ContractType: ast.TypeNode{Ident: "Logger"}}},
		},
	}
	importMap := map[string]string{"testmod/auth": "auth"}
	if err := ValidateModuleProviders("api", tc, importMap, perPkg); err != nil {
		t.Fatalf("ValidateModuleProviders: %v", err)
	}
}
