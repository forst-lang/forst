package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func TestCheckTypes_functionLiteral_captureAndCallback(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	src := `package main

func twice(fn func(Int): Int, x Int): Int {
	return fn(fn(x))
}

func main() {
	n := 10
	inc := func(x Int): Int { return x + 1 }
	addN := func(x Int): Int { return x + n }
	println(twice(inc, addN(3)))
}
`
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
}

func TestCheckTypes_deferGo_anonymousFunctionLiteral(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	src := `package main

func main() {
	go func() { println("go") }()
	defer func() { println("defer") }()
}
`
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
}
