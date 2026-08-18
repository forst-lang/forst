package gointerop_test

import (
	"errors"
	"testing"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/testutil"
	"forst/internal/typechecker/gointerop"
)

func TestCheckFuncCall_genericGoAPI_rejectsWithClearDiagnostic(t *testing.T) {
	t.Parallel()
	dir := testutil.ModuleRoot(t)
	loaded, err := goload.LoadByPkgPath(dir, []string{"slices"})
	if err != nil || len(loaded) == 0 {
		t.Skip("go/packages could not load slices")
	}
	pkgp := loaded["slices"]
	if pkgp == nil || pkgp.Types == nil {
		t.Skip("slices package types unavailable")
	}
	obj := pkgp.Types.Scope().Lookup("Contains")
	if obj == nil {
		t.Fatal("slices.Contains not found")
	}

	host := stubHost{}
	var gotCode, gotMsg string
	diag := func(_ ast.SourceSpan, code, format string, args ...any) error {
		gotCode = code
		gotMsg = format
		return errors.New("diag")
	}
	_, err = gointerop.CheckFuncCall(host, diag, gointerop.FuncCall{
		Pkg:      pkgp.Types,
		FuncName: "Contains",
		Call: ast.FunctionCallNode{
			Function: ast.Ident{ID: "Contains"},
		},
	})
	if err == nil {
		t.Fatal("expected error for generic Go API")
	}
	if gotCode != "go-call" {
		t.Fatalf("expected go-call diagnostic for generic API, got code=%q msg=%q", gotCode, gotMsg)
	}
}
