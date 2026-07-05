package parser

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestParseAssignment_indexLHS(t *testing.T) {
	t.Parallel()
	src := `package main

func f(arr []Int) {
	arr[0] = 2
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := assertNodeType[ast.FunctionNode](t, nodes[1], "ast.FunctionNode")
	assign := assertNodeType[ast.AssignmentNode](t, fn.Body[0], "ast.AssignmentNode")
	if _, ok := assign.LValues[0].(ast.IndexExpressionNode); !ok {
		t.Fatalf("lhs = %T", assign.LValues[0])
	}
}

func TestParseAssignment_shortOnIndexFails(t *testing.T) {
	t.Parallel()
	src := `package main

func f(arr []Int) {
	arr[0] := 1
}
`
	err := parseShouldFail(src)
	if err == nil || !strings.Contains(err.Error(), "cannot use :=") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseAssignment_compoundWithExplicitTypeFails(t *testing.T) {
	t.Parallel()
	src := `package main

func f() {
	var n: Int
	n: Int += 1
}
`
	err := parseShouldFail(src)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseMultipleAssignment_threeValues(t *testing.T) {
	t.Parallel()
	src := `package main

func f() {
	a, b = 1, 2, 3
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := assertNodeType[ast.FunctionNode](t, nodes[1], "ast.FunctionNode")
	assign := assertNodeType[ast.AssignmentNode](t, fn.Body[0], "ast.AssignmentNode")
	if len(assign.RValues) != 3 {
		t.Fatalf("rvalues = %d", len(assign.RValues))
	}
}
