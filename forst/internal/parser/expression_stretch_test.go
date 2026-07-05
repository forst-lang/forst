package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseExpression_stretchForms(t *testing.T) {
	t.Parallel()
	src := `package main

func run(x Int, y String): Bool {
	a := x + 1
	b := x - 1
	c := x * 2
	d := x / 2
	e := x % 3
	f := x > 1 && x < 10 || true
	g := len(y)
	h := &x
	i := (x + 1)
	j := x is GreaterThan(0)
	return f
}
`
	if _, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile(); err != nil {
		t.Fatal(err)
	}
}

func TestParseExpression_isKeywordAndSliceIndexChain(t *testing.T) {
	t.Parallel()
	src := `package main

func f(xs []Int, i Int): Int {
	return xs[i]
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := assertNodeType[ast.FunctionNode](t, nodes[1], "ast.FunctionNode")
	ret := assertNodeType[ast.ReturnNode](t, fn.Body[0], "ast.ReturnNode")
	if _, ok := ret.Values[0].(ast.IndexExpressionNode); !ok {
		t.Fatalf("return = %T", ret.Values[0])
	}
}

func TestParseBlockStatement_labeledContinue(t *testing.T) {
	t.Parallel()
	src := `package main

func f() {
loop:
	for true {
		continue loop
	}
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := assertNodeType[ast.FunctionNode](t, nodes[1], "ast.FunctionNode")
	loop := assertNodeType[*ast.ForNode](t, fn.Body[0], "*ast.ForNode")
	if loop.Label == nil || loop.Label.ID != "loop" {
		t.Fatalf("label = %v", loop.Label)
	}
	cont := assertNodeType[*ast.ContinueNode](t, loop.Body[0], "*ast.ContinueNode")
	if cont.Label == nil || cont.Label.ID != "loop" {
		t.Fatalf("continue label = %v", cont.Label)
	}
}

func TestParseFunction_pointerInBody(t *testing.T) {
	t.Parallel()
	src := `package main

func zero(): Int {
	x := 0
	p := &x
	return *p
}
`
	if _, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile(); err != nil {
		t.Fatal(err)
	}
}

func TestParseTypeDef_intersectionAndAssertion(t *testing.T) {
	t.Parallel()
	src := `package main

type U = Int | String
type Wrap = String.Min(1)
`
	if _, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile(); err != nil {
		t.Fatal(err)
	}
}

func TestParseEnsure_withBlock(t *testing.T) {
	t.Parallel()
	src := `package main

func f(x Int): Int {
	ensure x is GreaterThan(0) {
		return 0
	}
	return x
}
`
	if _, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile(); err != nil {
		t.Fatal(err)
	}
}
