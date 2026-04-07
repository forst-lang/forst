package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseFile_ensureWithOrClause_negativeIntConstraint(t *testing.T) {
	src := `package main

import "errors"

func invalidMove(msg String): Error {
	return errors.New(msg)
}

func f(row Int): Result(String, Error) {
	ensure row is GreaterThan(-1) or invalidMove("bad")
	return Ok("")
}
`
	log := ast.SetupTestLogger(nil)
	p := NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fn := findFunction(t, nodes, "f")
	ensure := fn.Body[0].(ast.EnsureNode)
	if len(ensure.Assertion.Constraints) != 1 {
		t.Fatalf("constraints: %+v", ensure.Assertion.Constraints)
	}
	if ensure.Assertion.Constraints[0].Name != "GreaterThan" {
		t.Fatalf("want GreaterThan, got %q", ensure.Assertion.Constraints[0].Name)
	}
	if ensure.Error == nil {
		t.Fatal("expected or clause on ensure")
	}
}

func TestParseFile_ensureConstraint_negativeIntArgument(t *testing.T) {
	src := `package main

func f(x Int): Int {
	ensure x is GreaterThan(-1)
	return x
}
`
	log := ast.SetupTestLogger(nil)
	p := NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fn := findFunction(t, nodes, "f")
	ensure := fn.Body[0].(ast.EnsureNode)
	if len(ensure.Assertion.Constraints) != 1 {
		t.Fatalf("constraints: %+v", ensure.Assertion.Constraints)
	}
	c := ensure.Assertion.Constraints[0]
	if c.Name != "GreaterThan" {
		t.Fatalf("constraint name: %s", c.Name)
	}
	if len(c.Args) != 1 || c.Args[0].Value == nil {
		t.Fatalf("args: %+v", c.Args)
	}
	il, ok := (*c.Args[0].Value).(ast.IntLiteralNode)
	if !ok || il.Value != -1 {
		t.Fatalf("want int literal -1, got %#v", c.Args[0].Value)
	}
}
