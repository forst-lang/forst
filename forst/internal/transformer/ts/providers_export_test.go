package transformerts

import (
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
