package gointerop_test

import (
	"errors"
	"strings"
	"testing"

	"go/types"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/testutil"
	"forst/internal/typechecker/gointerop"
)

func TestCheckFuncCall_genericGoAPI_instantiatesFromArgs(t *testing.T) {
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

	host := sliceContainsHost{}
	argTypes := [][]ast.TypeNode{
		{{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{{Ident: ast.TypeInt}}}},
		{{Ident: ast.TypeInt}},
	}
	var gotCode, gotMsg string
	diag := func(_ ast.SourceSpan, code, format string, args ...any) error {
		gotCode = code
		gotMsg = format
		return errors.New("diag")
	}
	_, err = gointerop.CheckFuncCall(host, diag, gointerop.FuncCall{
		Pkg:             pkgp.Types,
		FuncName:        "Contains",
		ArgTypes:        argTypes,
		RequireExported: true,
		Call: ast.FunctionCallNode{
			Function: ast.Ident{ID: "Contains"},
		},
	})
	if err != nil {
		t.Fatalf("expected slices.Contains to instantiate, got err=%v code=%q msg=%q", err, gotCode, gotMsg)
	}
}

type sliceContainsHost struct{}

func (sliceContainsHost) ForstTypeForGoType(_ types.Type) (ast.TypeNode, bool) {
	return ast.TypeNode{}, false
}
func (sliceContainsHost) IsTypeCompatible(_, _ ast.TypeNode) bool { return true }
func (sliceContainsHost) GoTypeForForstType(f ast.TypeNode) types.Type {
	if f.Ident == ast.TypeInt {
		return types.Typ[types.Int]
	}
	if f.Ident == ast.TypeArray && len(f.TypeParams) == 1 {
		elem := sliceContainsHost{}.GoTypeForForstType(f.TypeParams[0])
		if elem != nil {
			return types.NewSlice(elem)
		}
	}
	return nil
}
func (sliceContainsHost) InferExpressionType(_ ast.ExpressionNode) ([]ast.TypeNode, error) {
	return nil, nil
}

func (sliceContainsHost) GoTypeForExpression(_ ast.ExpressionNode) types.Type {
	return nil
}

func TestCheckFuncCall_unexportedSymbol_rejected(t *testing.T) {
	t.Parallel()
	pkg := types.NewPackage("p", "p")
	scope := pkg.Scope()
	fn := types.NewFunc(0, pkg, "secret", types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false))
	scope.Insert(fn)

	host := stubHost{}
	var gotMsg string
	diag := func(_ ast.SourceSpan, code, format string, args ...any) error {
		gotMsg = format
		return errors.New("diag")
	}
	_, err := gointerop.CheckFuncCall(host, diag, gointerop.FuncCall{
		Pkg:             pkg,
		FuncName:        "secret",
		RequireExported: true,
		Call:            ast.FunctionCallNode{Function: ast.Ident{ID: "secret"}},
	})
	if err == nil {
		t.Fatal("expected error for unexported symbol")
	}
	if !strings.Contains(gotMsg, "not exported") {
		t.Fatalf("expected unexported diagnostic, got %q", gotMsg)
	}
}
