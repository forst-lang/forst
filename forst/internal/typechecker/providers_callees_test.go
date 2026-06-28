package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestSplitQualifiedCallee(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in          string
		wantLocal   string
		wantFn      string
		wantOK      bool
	}{
		{"alpha.LogExpiry", "alpha", "LogExpiry", true},
		{"LogExpiry", "", "LogExpiry", false},
		{"", "", "", false},
		{"a.b", "a", "b", true},
	}
	for _, tc := range cases {
		local, fn, ok := splitQualifiedCallee(tc.in)
		if ok != tc.wantOK || local != tc.wantLocal || fn != tc.wantFn {
			t.Fatalf("splitQualifiedCallee(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tc.in, local, fn, ok, tc.wantLocal, tc.wantFn, tc.wantOK)
		}
	}
}

func TestProviderSlotsForCallee_intraPackage(t *testing.T) {
	tc := New(nil, false)
	tc.FunctionProviders = map[ast.Identifier][]ProviderSlot{
		"expireToken": {{RootIdent: "Logger", Key: "Logger"}},
	}
	slots := tc.providerSlotsForCallee("expireToken")
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("slots = %v", slots)
	}
}

func TestProviderSlotsForCallee_crossPackageSibling(t *testing.T) {
	alphaTC := New(nil, false)
	alphaTC.FunctionProviders = map[ast.Identifier][]ProviderSlot{
		"LogExpiry": {{RootIdent: "Logger", Key: "Logger"}},
	}
	view := &testModuleView{
		importMap: map[string]string{"demo/alpha": "alpha"},
		pkgs:      map[string]*TypeChecker{"alpha": alphaTC},
	}
	betaTC := New(nil, false)
	betaTC.SetModuleResult(view)
	betaTC.importPathByLocal = map[string]string{"alpha": "demo/alpha"}

	slots := betaTC.providerSlotsForCallee("alpha.LogExpiry")
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("cross-package slots = %v", slots)
	}
}

func TestProviderSlotsForCallee_crossPackageWithoutModuleReturnsNil(t *testing.T) {
	tc := New(nil, false)
	if slots := tc.providerSlotsForCallee("alpha.LogExpiry"); slots != nil {
		t.Fatalf("expected nil without module, got %v", slots)
	}
}

func TestMergeModuleKnownRoots_unionsAcrossPackages(t *testing.T) {
	alpha := New(nil, false)
	alpha.providersEngine().KnownRoots["Logger"] = ast.TypeNode{Ident: "Logger"}
	beta := New(nil, false)
	beta.providersEngine().KnownRoots["Clock"] = ast.TypeNode{Ident: "Clock"}

	MergeModuleKnownRoots(map[string]*TypeChecker{"alpha": alpha, "beta": beta})

	for _, tc := range []*TypeChecker{alpha, beta} {
		if _, ok := tc.providers.KnownRoots["Logger"]; !ok {
			t.Fatal("expected Logger in KnownRoots after merge")
		}
		if _, ok := tc.providers.KnownRoots["Clock"]; !ok {
			t.Fatal("expected Clock in KnownRoots after merge")
		}
	}
}

func TestRevalidateUnusedWiringKeysAfterModuleMerge_stripsStaleAndRecomputes(t *testing.T) {
	tc := New(nil, false)
	tc.providersEngine().PendingWith = []pendingWithCheck{
		{
			with: ast.WithNode{
				Wiring: ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"Logger": {Node: ast.NilLiteralNode{}},
						"Clock":  {Node: ast.NilLiteralNode{}},
					},
				},
				Body: []ast.Node{
					ast.FunctionCallNode{
						Function: ast.Ident{ID: "expireToken"},
					},
				},
			},
			innerKeys: map[string]struct{}{"Logger": {}, "Clock": {}},
		},
	}
	tc.Warnings = []Diagnostic{
		{Code: "providers-unused-key", Msg: "wiring key \"Logger\" is not required"},
		{Code: "other-warning", Msg: "keep me"},
	}
	tc.FunctionProviders = map[ast.Identifier][]ProviderSlot{
		"expireToken": {{RootIdent: "Logger", Key: "Logger"}},
	}

	tc.RevalidateUnusedWiringKeysAfterModuleMerge()

	for _, w := range tc.Warnings {
		if w.Code == "providers-unused-key" && strings.Contains(w.Msg, "Logger") {
			t.Fatalf("stale Logger unused warning should be stripped, got: %s", w.Msg)
		}
	}
	foundClock := false
	foundOther := false
	for _, w := range tc.Warnings {
		if w.Code == "providers-unused-key" && strings.Contains(w.Msg, "Clock") {
			foundClock = true
		}
		if w.Code == "other-warning" {
			foundOther = true
		}
	}
	if !foundClock {
		t.Fatal("expected recomputed unused Clock warning")
	}
	if !foundOther {
		t.Fatal("expected non-unused warnings preserved")
	}
}

func TestRevalidateDeferredWiringKeysAfterModuleMerge_crossPackageKeys(t *testing.T) {
	host := New(nil, false)
	host.providersEngine().DeferWiringRootCheck = true
	host.providersEngine().PendingWith = []pendingWithCheck{
		{
			with: ast.WithNode{
				Wiring: ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"Logger": {Node: ast.NilLiteralNode{}},
					},
				},
			},
		},
	}
	lib := New(nil, false)
	lib.providersEngine().KnownRoots["Logger"] = ast.TypeNode{Ident: "Logger"}

	MergeModuleKnownRoots(map[string]*TypeChecker{"host": host, "lib": lib})
	if err := host.RevalidateDeferredWiringKeysAfterModuleMerge(); err != nil {
		t.Fatalf("revalidate: %v", err)
	}
}

func TestRevalidateDeferredWiringKeysAfterModuleMerge_unknownKeyStillErrors(t *testing.T) {
	tc := New(nil, false)
	tc.providersEngine().DeferWiringRootCheck = true
	tc.providersEngine().PendingWith = []pendingWithCheck{
		{
			with: ast.WithNode{
				Wiring: ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"UnknownKey": {Node: ast.NilLiteralNode{}},
					},
				},
			},
		},
	}
	if err := tc.RevalidateDeferredWiringKeysAfterModuleMerge(); err == nil {
		t.Fatal("expected unknown wiring key error after revalidation")
	}
}

type testModuleView struct {
	importMap map[string]string
	pkgs      map[string]*TypeChecker
}

func (v *testModuleView) ImportPathToForstPkg() map[string]string { return v.importMap }
func (v *testModuleView) ForstPackageTypeChecker(pkg string) *TypeChecker { return v.pkgs[pkg] }
