package gointerop_test

import (
	"testing"

	"go/types"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/testutil"
	"forst/internal/typechecker/gointerop"
)

func TestInstantiateFuncSignature_slicesContains(t *testing.T) {
	t.Parallel()
	dir := testutil.ModuleRoot(t)
	loaded, err := goload.LoadByPkgPath(dir, []string{"slices"})
	if err != nil || len(loaded) == 0 {
		t.Skip("go/packages could not load slices")
	}
	fn := loaded["slices"].Types.Scope().Lookup("Contains").(*types.Func)
	argGo := []types.Type{types.NewSlice(types.Typ[types.Int]), types.Typ[types.Int]}
	sig, err := gointerop.InstantiateFuncSignature(fn, argGo)
	if err != nil {
		t.Fatalf("InstantiateFuncSignature: %v", err)
	}
	if sig.Params().Len() != 2 {
		t.Fatalf("params len=%d", sig.Params().Len())
	}
	_ = sig
}

func TestGoTypesFromForstArgs_sliceInt(t *testing.T) {
	t.Parallel()
	host := sliceContainsHost{}
	args := [][]ast.TypeNode{
		{{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{{Ident: ast.TypeInt}}}},
		{{Ident: ast.TypeInt}},
	}
	got := gointerop.GoTypesFromForstArgs(host, args)
	if len(got) != 2 || got[0] == nil || got[1] == nil {
		t.Fatalf("GoTypesFromForstArgs: %#v", got)
	}
}
