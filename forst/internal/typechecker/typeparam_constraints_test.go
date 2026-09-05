package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/testutil"
)

func TestValidateFunctionTypeParamConstraints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		src         string
		expectError string
	}{
		{
			name: "unknownConstraint",
			src: `package main

func bad[T ordered](x T): T { return x }

func main() { _ = bad(1) }
`,
			expectError: `unknown constraint "ordered"`,
		},
		{
			name: "comparableRejectsSlice",
			src: `package main

func eq[T comparable](a T, b T): Bool { return a == b }

func main() { _ = eq([1], [1]) }
`,
			expectError: "does not satisfy comparable constraint",
		},
		{
			name: "comparableAllowsInt",
			src: `package main

func eq[T comparable](a T, b T): Bool { return a == b }

func main() {
	x := eq(1, 2)
	println(x)
}
`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := testutil.TypecheckOpts{FileID: "typeparam_constraints.ft"}
			if tt.expectError != "" {
				opts.ExpectError = tt.expectError
			}
			MustTypecheck(t, tt.src, opts)
		})
	}
}

func TestIsComparableForstType(t *testing.T) {
	t.Parallel()
	tc := New(testutil.TestLogger(t, nil), false)

	tests := []struct {
		name string
		typ  ast.TypeNode
		want bool
	}{
		{name: "int", typ: ast.NewBuiltinType(ast.TypeInt), want: true},
		{name: "string", typ: ast.NewBuiltinType(ast.TypeString), want: true},
		{name: "slice", typ: ast.NewArrayType(ast.NewBuiltinType(ast.TypeInt)), want: false},
		{name: "fixedArray", typ: ast.NewFixedArrayType(ast.NewBuiltinType(ast.TypeInt), 2), want: true},
		{name: "map", typ: ast.NewMapType(ast.NewBuiltinType(ast.TypeString), ast.NewBuiltinType(ast.TypeInt)), want: false},
		{name: "typeParam", typ: ast.NewTypeParamType("T"), want: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.isComparableForstType(tt.typ); got != tt.want {
				t.Fatalf("isComparableForstType(%s) = %v, want %v", tt.typ.String(), got, tt.want)
			}
		})
	}
}
