package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/testutil"
)

func TestNormalizeGoImportParamType_colonTestingT(t *testing.T) {
	t.Parallel()
	tc := testTypeChecker(t)
	got, ok := tc.normalizeGoImportParamType(ast.TypeNode{Ident: ast.TypeIdent("testing.T")})
	if !ok {
		t.Fatal("expected testing.T to normalize")
	}
	if got.Ident != ast.TypePointer || len(got.TypeParams) != 1 || got.TypeParams[0].Ident != ast.TypeIdent("testing.T") {
		t.Fatalf("got %+v, want Pointer(testing.T)", got)
	}
}

func TestCheckTypes_testFunctionColonTestingT_keepsPointerForm(t *testing.T) {
	t.Parallel()
	src := `package main

import "testing"

func TestEnsureOnly(t:testing.T) {}
`
	tc, nodes := MustTypecheck(t, src, testutil.TypecheckOpts{})
	var fn ast.FunctionNode
	for _, n := range nodes {
		if f, ok := n.(ast.FunctionNode); ok && f.Ident.ID == "TestEnsureOnly" {
			fn = f
			break
		}
	}
	if !tc.IsGoTestFunction(fn) {
		t.Fatal("expected IsGoTestFunction")
	}
	sig := tc.Functions["TestEnsureOnly"]
	if len(sig.Parameters) != 1 || !ast.IsTestingTParamType(sig.Parameters[0].Type) {
		t.Fatalf("signature param type = %+v, want Pointer(testing.T)", sig.Parameters[0].Type)
	}
}
