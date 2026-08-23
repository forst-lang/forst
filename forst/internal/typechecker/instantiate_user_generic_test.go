package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

func TestInstantiateGenericCall_identity(t *testing.T) {
	t.Parallel()
	src := `package main

func identity[T any](x T): T {
	return x
}

func main() {
	x := identity(42)
	println(string(x))
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{FileID: "generic_function.ft"})
}

func TestInstantiateGenericCall_firstFromSlice(t *testing.T) {
	t.Parallel()
	src := `package main

func first[T any](xs Array(T)): T {
	return xs[0]
}

func main() {
	n := first([1, 2, 3])
	println(string(n))
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{FileID: "generic_first.ft"})
}

func TestInstantiateGenericCall_twoTypeParams(t *testing.T) {
	t.Parallel()
	src := `package main

func pick[T any, U any](a T, b U): T {
	return a
}

func main() {
	n := pick(1, "x")
	println(string(n))
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{FileID: "generic_pick.ft"})
}

func TestInstantiateGenericCall_unifyConflict(t *testing.T) {
	t.Parallel()
	src := `package main

func bad[T any](a T, b T): T {
	return a
}

func main() {
	_ = bad(1, "x")
}
`
	_, _, err := Typecheck(t, src, testutil.TypecheckOpts{FileID: "bad.ft"})
	if err == nil {
		t.Fatal("expected type error for conflicting type argument inference")
	}
}

func TestIsTypeParamType(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	tc.Defs[ast.TypeIdent("T")] = ast.TypeNode{Ident: "T", TypeKind: ast.TypeKindTypeParam}
	if !tc.IsTypeParamType(ast.TypeNode{Ident: "T"}) {
		t.Fatal("expected T to be a type parameter")
	}
	if tc.IsTypeParamType(ast.TypeNode{Ident: ast.TypeInt}) {
		t.Fatal("Int must not be a type parameter")
	}
}
