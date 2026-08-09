package transformerts

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

func TestShouldEmitFunctionToTypeScript(t *testing.T) {
	tc := typechecker.New(nil, false)
	if tc.FunctionProviders == nil {
		tc.FunctionProviders = make(map[ast.Identifier][]typechecker.ProviderSlot)
	}
	tc.FunctionProviders["Needs"] = []typechecker.ProviderSlot{{RootIdent: "Logger"}}

	runnable := ast.FunctionNode{Ident: ast.Ident{ID: "Echo"}}
	needs := ast.FunctionNode{Ident: ast.Ident{ID: "Needs"}}
	private := ast.FunctionNode{Ident: ast.Ident{ID: "helper"}}

	if !ShouldEmitFunctionToTypeScript(runnable, tc) {
		t.Fatal("runnable public fn should emit")
	}
	if ShouldEmitFunctionToTypeScript(needs, tc) {
		t.Fatal("public fn with Providers should not emit")
	}
	if ShouldEmitFunctionToTypeScript(private, tc) {
		t.Fatal("private fn should not emit to TS")
	}
}

func TestShouldEmitFunctionToTypeScript_nilTypeCheckerAndReceiver(t *testing.T) {
	publicFn := ast.FunctionNode{Ident: ast.Ident{ID: "Echo"}}
	if !ShouldEmitFunctionToTypeScript(publicFn, nil) {
		t.Fatal("public function should emit when typechecker is nil")
	}

	methodFn := ast.FunctionNode{
		Ident: ast.Ident{ID: "Method"},
		Receiver: &ast.SimpleParamNode{
			Ident: ast.Ident{ID: "r"},
			Type:  ast.NewBuiltinType(ast.TypeString),
		},
	}
	if ShouldEmitFunctionToTypeScript(methodFn, nil) {
		t.Fatal("receiver methods should not emit")
	}
}

func TestProviderOmissionReason_namesUnsatisfiedProvider(t *testing.T) {
	tc := typechecker.New(nil, false)
	tc.FunctionProviders = map[ast.Identifier][]typechecker.ProviderSlot{
		"Login": {{RootIdent: "db", Key: "db"}},
	}
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "Login"}}
	reason, emit := ProviderOmissionReason(fn, tc)
	if emit {
		t.Fatal("expected omission")
	}
	if reason != `provider "db" not satisfied` {
		t.Fatalf("reason = %q", reason)
	}
}

func TestProviderOmissionReason_multipleProviders(t *testing.T) {
	tc := typechecker.New(nil, false)
	tc.FunctionProviders = map[ast.Identifier][]typechecker.ProviderSlot{
		"Register": {
			{RootIdent: "Clock", Key: "Clock"},
			{RootIdent: "Logger", Key: "Logger"},
		},
	}
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "Register"}}
	reason, emit := ProviderOmissionReason(fn, tc)
	if emit {
		t.Fatal("expected omission")
	}
	if !strings.Contains(reason, "providers ") || !strings.Contains(reason, `"Clock"`) || !strings.Contains(reason, `"Logger"`) {
		t.Fatalf("reason = %q", reason)
	}
}

func TestCollectOmittedFunctions_listsProviderGatedOnly(t *testing.T) {
	tc := typechecker.New(nil, false)
	tc.FunctionProviders = map[ast.Identifier][]typechecker.ProviderSlot{
		"Login":    {{RootIdent: "db", Key: "db"}},
		"Register": {{RootIdent: "db", Key: "db"}},
	}
	nodes := []ast.Node{
		ast.PackageNode{Ident: ast.Ident{ID: "auth"}},
		ast.FunctionNode{Ident: ast.Ident{ID: "Echo"}},
		ast.FunctionNode{Ident: ast.Ident{ID: "helper"}},
		ast.FunctionNode{Ident: ast.Ident{ID: "Login"}},
		ast.FunctionNode{Ident: ast.Ident{ID: "Register"}},
	}
	got := CollectOmittedFunctions("auth", nodes, tc)
	if len(got) != 2 {
		t.Fatalf("got %d omissions, want 2: %#v", len(got), got)
	}
	if got[0].FunctionName != "Login" || got[1].FunctionName != "Register" {
		t.Fatalf("order/names = %#v", got)
	}
	for _, o := range got {
		if o.PackageName != "auth" {
			t.Fatalf("package = %q", o.PackageName)
		}
		if o.Reason != `provider "db" not satisfied` {
			t.Fatalf("reason = %q", o.Reason)
		}
	}
}
