package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func TestArrayLength_fixedArrayNotAssignableToSlice(t *testing.T) {
	t.Parallel()
	tc := New(setupTestLogger(nil), false)
	slice := ast.NewArrayType(ast.NewBuiltinType(ast.TypeInt))
	fixed := ast.NewFixedArrayType(ast.NewBuiltinType(ast.TypeInt), 3)
	if tc.IsTypeCompatible(fixed, slice) {
		t.Fatal("fixed array must not assign to slice")
	}
	if tc.IsTypeCompatible(slice, fixed) {
		t.Fatal("slice must not assign to fixed array")
	}
}

func TestArrayLength_sameFixedLengthAssignable(t *testing.T) {
	t.Parallel()
	tc := New(setupTestLogger(nil), false)
	a := ast.NewFixedArrayType(ast.NewBuiltinType(ast.TypeInt), 3)
	b := ast.NewFixedArrayType(ast.NewBuiltinType(ast.TypeInt), 3)
	if !tc.IsTypeCompatible(a, b) {
		t.Fatal("same [3]Int should be compatible")
	}
	c := ast.NewFixedArrayType(ast.NewBuiltinType(ast.TypeInt), 4)
	if tc.IsTypeCompatible(a, c) {
		t.Fatal("[3]Int must not match [4]Int")
	}
}

func TestArrayLength_appendRejectsFixedArray(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	var xs [3]Int
	append(xs, 1)
}
`
	log := setupTestLogger(nil)
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil || !strings.Contains(err.Error(), "slice") {
		t.Fatalf("got %v", err)
	}
}

func TestArrayLength_literalLengthMismatch(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	var xs [3]Int = [1, 2]
}
`
	log := setupTestLogger(nil)
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil || !strings.Contains(err.Error(), "2 elements") {
		t.Fatalf("got %v", err)
	}
}

func TestArrayLength_indexOutOfBounds(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	var xs [3]Int
	println(xs[3])
}
`
	log := setupTestLogger(nil)
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("got %v", err)
	}
}
