package transformerts

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

func TestShouldEmitFunctionToTypeScript(t *testing.T) {
	tc := typechecker.New(nil, false)
	if tc.FunctionUsables == nil {
		tc.FunctionUsables = make(map[ast.Identifier][]typechecker.UsableSlot)
	}
	tc.FunctionUsables["Needs"] = []typechecker.UsableSlot{{RootIdent: "Logger"}}

	runnable := ast.FunctionNode{Ident: ast.Ident{ID: "Echo"}}
	needs := ast.FunctionNode{Ident: ast.Ident{ID: "Needs"}}
	private := ast.FunctionNode{Ident: ast.Ident{ID: "helper"}}

	if !ShouldEmitFunctionToTypeScript(runnable, tc) {
		t.Fatal("runnable public fn should emit")
	}
	if ShouldEmitFunctionToTypeScript(needs, tc) {
		t.Fatal("public fn with Usables should not emit")
	}
	if ShouldEmitFunctionToTypeScript(private, tc) {
		t.Fatal("private fn should not emit to TS")
	}
}
