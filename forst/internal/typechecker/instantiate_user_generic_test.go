package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/testutil"
)

func TestInstantiateGenericCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		src         string
		fileID      string
		expectError string
		check       func(t *testing.T, tc *TypeChecker)
	}{
		{
			name: "callResultType",
			src: `package main

func identity[T any](x T): T {
	return x
}

func main() {
	x := identity(42)
	if x != 0 {
		println(string(x))
	}
}
`,
			fileID: "generic_function.ft",
			check: func(t *testing.T, tc *TypeChecker) {
				t.Helper()
				vt := tc.VariableTypes[ast.Identifier("x")]
				if len(vt) != 1 || vt[0].Ident != ast.TypeInt {
					t.Fatalf("x type = %+v, want Int", vt)
				}
			},
		},
		{
			name: "identity",
			src: `package main

func identity[T any](x T): T {
	return x
}

func main() {
	x := identity(42)
	println(string(x))
}
`,
			fileID: "generic_function.ft",
		},
		{
			name: "firstFromSlice",
			src: `package main

func first[T any](xs Array(T)): T {
	return xs[0]
}

func main() {
	n := first([1, 2, 3])
	println(string(n))
}
`,
			fileID: "generic_first.ft",
		},
		{
			name: "twoTypeParams",
			src: `package main

func pick[T any, U any](a T, b U): T {
	return a
}

func main() {
	n := pick(1, "x")
	println(string(n))
}
`,
			fileID: "generic_pick.ft",
		},
		{
			name: "unifyConflict",
			src: `package main

func bad[T any](a T, b T): T {
	return a
}

func main() {
	_ = bad(1, "x")
}
`,
			fileID:      "bad.ft",
			expectError: "inferred as",
		},
		{
			name: "comparableRejectsSlice",
			src: `package main

func eq[T comparable](a T, b T): Bool {
	return a == b
}

func main() {
	var discard Bool
	discard = eq([1], [1])
	println(discard)
}
`,
			fileID:      "generic_eq.ft",
			expectError: "does not satisfy comparable constraint",
		},
		{
			name: "explicitTypeArgs",
			src: `package main

func identity[T any](x T): T {
	return x
}

func main() {
	n := identity[Int](42)
	println(string(n))
}
`,
			fileID: "generic_function_explicit.ft",
		},
		{
			name: "pointerParam",
			src: `package main

func deref[T any](p *T): T {
	return *p
}

func main() {
	n := 7
	x := deref(&n)
	println(string(x))
}
`,
			fileID: "generic_pointer.ft",
		},
		{
			name: "mapTwoTypeParams",
			src: `package main

func mapLen[K comparable, V any](m map[K]V): Int {
	return len(m)
}

func main() {
	m := map[String]Int{"a": 1}
	println(string(mapLen(m)))
}
`,
			fileID: "generic_map.ft",
		},
		{
			name: "fixedArrayLengthMismatch",
			src: `package main

func takeTwo[T any](xs: [2]T): T {
	return xs[0]
}

func main() {
	_ = takeTwo([1, 2, 3])
}
`,
			fileID:      "generic_fixed_array.ft",
			expectError: "array literal has 3 elements",
		},
		{
			name: "explicitTypeArgConflict",
			src: `package main

func identity[T any](x T): T {
	return x
}

func main() {
	_ = identity[String](42)
}
`,
			fileID:      "generic_explicit_conflict.ft",
			expectError: "inferred as",
		},
		{
			name: "variadicAllSame",
			src: `package main

func ignore[T any](xs ...T): Bool {
	return true
}

func main() {
	println(ignore(1, 2, 3))
}
`,
			fileID: "generic_variadic.ft",
		},
		{
			name: "genericShapeFieldAccess",
			src: `package main

func getValue[T any](b { value: T }): T {
	return b.value
}

func main() {
	println(string(getValue({ value: 42 })))
}
`,
			fileID: "generic_shape.ft",
		},
		{
			name: "genericResultParam",
			src: `package main

func one() {
	n := 1
	ensure n is GreaterThan(0)
	return n
}

func accept[T any](r Result(T, Error)): Bool {
	return true
}

func main() {
	println(accept(one()))
}
`,
			fileID: "generic_result.ft",
		},
		{
			name: "unboundTypeParamHint",
			src: `package main

func empty[T any](): Bool {
	return true
}

func main() {
	empty()
}
`,
			fileID:      "generic_shape_unbound.ft",
			expectError: "try explicit type arguments",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := testutil.TypecheckOpts{FileID: tt.fileID}
			if tt.expectError != "" {
				opts.ExpectError = tt.expectError
			}
			tc, _ := MustTypecheck(t, tt.src, opts)
			if tt.check != nil {
				tt.check(t, tc)
			}
		})
	}
}

func TestIsTypeParamType(t *testing.T) {
	t.Parallel()
	tc := New(testutil.TestLogger(t, nil), false)
	if !tc.IsTypeParamType(ast.NewTypeParamType("T")) {
		t.Fatal("expected T to be a type parameter")
	}
	if tc.IsTypeParamType(ast.NewBuiltinType(ast.TypeInt)) {
		t.Fatal("Int must not be a type parameter")
	}
}
