package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseAssertion_builtinFloatAndBoolBaseTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		src  string
		want ast.TypeIdent
	}{
		{src: `Float.Present()`, want: ast.TypeFloat},
		{src: `Bool.True()`, want: ast.TypeBool},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(string(tc.want), func(t *testing.T) {
			t.Parallel()
			p := NewTestParser(tc.src, ast.SetupTestLogger(nil))
			assertion := p.parseAssertionChain(true)
			if assertion.BaseType == nil || *assertion.BaseType != tc.want {
				t.Fatalf("base = %v want %v", assertion.BaseType, tc.want)
			}
		})
	}
}

func TestParseAssertion_constraintWithTypeArgument(t *testing.T) {
	t.Parallel()
	p := NewTestParser(`Between(1, Int)`, ast.SetupTestLogger(nil))
	assertion := p.parseAssertionChain(false)
	if len(assertion.Constraints) != 1 {
		t.Fatalf("constraints = %+v", assertion.Constraints)
	}
	if len(assertion.Constraints[0].Args) != 2 || assertion.Constraints[0].Args[1].Type == nil {
		t.Fatalf("args = %+v", assertion.Constraints[0].Args)
	}
}

func TestParseExpression_unaryBinaryCallAndReference(t *testing.T) {
	t.Parallel()
	src := `package main

func f(x Int, arr []Int): Int {
	a := &x
	b := arr[0]
	c := len(arr)
	d := !true
	return a + b + c + d
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := assertNodeType[ast.FunctionNode](t, nodes[1], "ast.FunctionNode")
	if len(fn.Body) < 5 {
		t.Fatalf("body len = %d", len(fn.Body))
	}
}

func TestParseFor_rangeAndLabeledBreak(t *testing.T) {
	t.Parallel()
	src := `package main

func sum(xs []Int): Int {
outer:
	for range xs {
		break outer
	}
	return 0
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := assertNodeType[ast.FunctionNode](t, nodes[1], "ast.FunctionNode")
	forNode := assertNodeType[*ast.ForNode](t, fn.Body[0], "*ast.ForNode")
	if forNode.Label == nil || forNode.Label.ID != "outer" {
		t.Fatalf("label = %v", forNode.Label)
	}
}

func TestParseFunction_destructuredParamAndReceiver(t *testing.T) {
	t.Parallel()
	src := `package main

type Box = { value: Int }

func (b Box) swap({a, b}: { a: Int, b: Int }): Int {
	return a
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := assertNodeType[ast.FunctionNode](t, nodes[2], "ast.FunctionNode")
	if fn.Receiver == nil || len(fn.Params) != 1 {
		t.Fatalf("receiver=%v params=%d", fn.Receiver, len(fn.Params))
	}
}
