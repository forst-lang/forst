package gointerop_test

import (
	"testing"

	"go/types"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/testutil"
	"forst/internal/typechecker/gointerop"
)

func loadSlicesFunc(t *testing.T, name string) *types.Func {
	t.Helper()
	dir := testutil.ModuleRoot(t)
	loaded, err := goload.LoadByPkgPath(dir, []string{"slices"})
	if err != nil || len(loaded) == 0 {
		t.Skip("go/packages could not load slices")
	}
	obj := loaded["slices"].Types.Scope().Lookup(name)
	if obj == nil {
		t.Fatalf("slices.%s not found", name)
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		t.Fatalf("slices.%s is not a func", name)
	}
	return fn
}

func TestInstantiateFuncSignature_slicesContains(t *testing.T) {
	t.Parallel()
	fn := loadSlicesFunc(t, "Contains")
	argGo := []types.Type{types.NewSlice(types.Typ[types.Int]), types.Typ[types.Int]}
	sig, err := gointerop.InstantiateFuncSignature(fn, argGo)
	if err != nil {
		t.Fatalf("InstantiateFuncSignature: %v", err)
	}
	if sig.Params().Len() != 2 {
		t.Fatalf("params len=%d", sig.Params().Len())
	}
	sliceParam := sig.Params().At(0).Type()
	intParam := sig.Params().At(1).Type()
	if _, ok := sliceParam.Underlying().(*types.Slice); !ok {
		t.Fatalf("param 0 = %v, want slice", sliceParam)
	}
	if sliceParam.Underlying().(*types.Slice).Elem() != types.Typ[types.Int] {
		t.Fatalf("slice elem = %v, want int", sliceParam.Underlying().(*types.Slice).Elem())
	}
	if intParam != types.Typ[types.Int] {
		t.Fatalf("param 1 = %v, want int", intParam)
	}
	if sig.TypeParams() != nil && sig.TypeParams().Len() > 0 {
		t.Fatalf("expected zero type params after instantiation, got %d", sig.TypeParams().Len())
	}
}

func TestInstantiateFuncSignature_slicesSort_multiTypeParam(t *testing.T) {
	t.Parallel()
	fn := loadSlicesFunc(t, "Sort")
	argGo := []types.Type{types.NewSlice(types.Typ[types.Int])}
	sig, err := gointerop.InstantiateFuncSignature(fn, argGo)
	if err != nil {
		t.Fatalf("InstantiateFuncSignature Sort: %v", err)
	}
	if sig.Params().Len() != 1 {
		t.Fatalf("params len=%d", sig.Params().Len())
	}
	param := sig.Params().At(0).Type()
	sl, ok := param.Underlying().(*types.Slice)
	if !ok {
		t.Fatalf("param = %v, want []int", param)
	}
	if sl.Elem() != types.Typ[types.Int] {
		t.Fatalf("slice elem = %v, want int", sl.Elem())
	}
	if sig.TypeParams() != nil && sig.TypeParams().Len() > 0 {
		t.Fatalf("expected zero type params after Sort instantiation, got %d", sig.TypeParams().Len())
	}
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
