package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

func TestInstantiateGenericCall_callResultType(t *testing.T) {
	t.Parallel()
	src := `package main

func identity[T any](x T): T {
	return x
}

func main() {
	x := identity(42)
	if x != 0 {
		println(string(x))
	}
}
`
	tc, _, err := Typecheck(t, src, testutil.TypecheckOpts{FileID: "generic_function.ft"})
	if err != nil {
		t.Fatal(err)
	}
	vt := tc.VariableTypes[ast.Identifier("x")]
	if len(vt) != 1 || vt[0].Ident != ast.TypeInt {
		t.Fatalf("x type = %+v, want Int", vt)
	}
}

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
	if !tc.IsTypeParamType(ast.NewTypeParamType("T")) {
		t.Fatal("expected T to be a type parameter")
	}
	if tc.IsTypeParamType(ast.NewBuiltinType(ast.TypeInt)) {
		t.Fatal("Int must not be a type parameter")
	}
}

func TestInstantiateGenericCall_comparableRejectsSlice(t *testing.T) {
	t.Parallel()
	src := `package main

func eq[T comparable](a T, b T): Bool {
	return a == b
}

func main() {
	_ = eq([1], [1])
}
`
	_, _, err := Typecheck(t, src, testutil.TypecheckOpts{FileID: "generic_eq.ft"})
	if err == nil {
		t.Fatal("expected comparable constraint error for slice arguments")
	}
}

func TestInstantiateGenericCall_explicitTypeArgs(t *testing.T) {
	t.Parallel()
	src := `package main

func identity[T any](x T): T {
	return x
}

func main() {
	n := identity[Int](42)
	println(string(n))
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{FileID: "generic_function_explicit.ft"})
}
