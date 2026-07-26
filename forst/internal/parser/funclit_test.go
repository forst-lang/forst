package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseFunctionLiteral_basicAndCall(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	f := func(x Int): Int { return x }
	_ = func(): Int { return 1 }()
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	mainFn := nodes[1].(ast.FunctionNode)
	assign := mainFn.Body[0].(ast.AssignmentNode)
	lit, ok := assign.RValues[0].(ast.FunctionLiteralNode)
	if !ok {
		t.Fatalf("rhs = %T, want FunctionLiteralNode", assign.RValues[0])
	}
	if len(lit.Params) != 1 || len(lit.ReturnTypes) != 1 {
		t.Fatalf("literal params=%d returns=%d", len(lit.Params), len(lit.ReturnTypes))
	}
	callAssign := mainFn.Body[1].(ast.AssignmentNode)
	call, ok := callAssign.RValues[0].(ast.FunctionCallNode)
	if !ok || call.Callee == nil {
		t.Fatalf("iife = %T callee=%v", callAssign.RValues[0], call.Callee)
	}
}

func TestParseFunctionType_parameter(t *testing.T) {
	t.Parallel()
	src := `package main

func apply(fn func(Int): Int, x Int): Int {
	return fn(x)
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := nodes[1].(ast.FunctionNode)
	sp := fn.Params[0].(ast.SimpleParamNode)
	if !sp.Type.IsFunctionType() {
		t.Fatalf("fn param type = %s", sp.Type.String())
	}
}
