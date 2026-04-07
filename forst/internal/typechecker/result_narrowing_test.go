package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"

)

func TestIfResult_isOk_narrowsSuccessType(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func f(): Result(Int, Error) {
	return Ok(0)
}

func main() {
	x := f()
	if x is Ok() {
		y := x
		println(y)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	v := ast.VariableNode{Ident: ast.Ident{ID: "y"}}
	types, ok := tc.InferredTypesForVariableNode(v)
	if !ok || len(types) != 1 {
		t.Fatalf("y: ok=%v types=%v", ok, types)
	}
	if types[0].Ident != ast.TypeInt {
		t.Fatalf("expected y narrowed to Int, got %s", types[0].String())
	}
}

func TestIfResult_isOk_wrongSubject_errors(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func main() {
	x := 1
	if x is Ok() {
		println(x)
	}
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected CheckTypes error: Ok() requires Result subject")
	}
	if !strings.Contains(err.Error(), "Result") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureResult_isOk_validates(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func f(): Result(Int, Error) {
	return Ok(0)
}

func main() {
	x := f()
	ensure x is Ok(0)
	println(x)
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}
