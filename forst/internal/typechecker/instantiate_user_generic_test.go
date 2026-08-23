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
