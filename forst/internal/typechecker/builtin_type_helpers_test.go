package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestBuiltinHelpers_lenOperandAllowed(t *testing.T) {
	tests := []struct {
		name string
		typ  ast.TypeNode
		want bool
	}{
		{"string", ast.NewBuiltinType(ast.TypeString), true},
		{"map", ast.NewMapType(ast.NewBuiltinType(ast.TypeString), ast.NewBuiltinType(ast.TypeInt)), true},
		{"slice", ast.NewArrayType(ast.NewBuiltinType(ast.TypeInt)), true},
		{"pointer_to_slice", ast.NewPointerType(ast.NewArrayType(ast.NewBuiltinType(ast.TypeBool))), true},
		{"implicit_go", ast.NewBuiltinType(ast.TypeImplicit), true},
		{"int", ast.NewBuiltinType(ast.TypeInt), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lenOperandAllowed(tt.typ); got != tt.want {
				t.Fatalf("lenOperandAllowed(%v) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestBuiltinHelpers_capOperandAllowed(t *testing.T) {
	tests := []struct {
		name string
		typ  ast.TypeNode
		want bool
	}{
		{"slice", ast.NewArrayType(ast.NewBuiltinType(ast.TypeInt)), true},
		{"pointer_to_slice", ast.NewPointerType(ast.NewArrayType(ast.NewBuiltinType(ast.TypeBool))), true},
		{"implicit_go", ast.NewBuiltinType(ast.TypeImplicit), true},
		{"string", ast.NewBuiltinType(ast.TypeString), false},
		{"map", ast.NewMapType(ast.NewBuiltinType(ast.TypeString), ast.NewBuiltinType(ast.TypeInt)), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := capOperandAllowed(tt.typ); got != tt.want {
				t.Fatalf("capOperandAllowed(%v) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestBuiltinHelpers_builtinTypesIdenticalOrdered(t *testing.T) {
	a := ast.NewArrayType(ast.NewBuiltinType(ast.TypeInt))
	b := ast.NewArrayType(ast.NewBuiltinType(ast.TypeInt))
	c := ast.NewArrayType(ast.NewBuiltinType(ast.TypeString))
	if !builtinTypesIdenticalOrdered(a, b) {
		t.Fatal("expected identical")
	}
	if builtinTypesIdenticalOrdered(a, c) {
		t.Fatal("expected not identical")
	}
}
