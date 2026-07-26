package parser

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestParse_switch_tagForm(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	switch x := 1; x {
	case 1, 2:
		println("small")
	case 3:
		println("three")
	default:
		println("other")
	}
}
`
	p := NewTestParser(src, ast.SetupTestLogger(nil))
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	mainFn := nodes[len(nodes)-1].(ast.FunctionNode)
	sw := assertNodeType[*ast.SwitchNode](t, mainFn.Body[0], "*ast.SwitchNode")
	if sw.Tag == nil {
		t.Fatal("expected tag expression")
	}
	if sw.Init == nil {
		t.Fatal("expected init statement")
	}
	if len(sw.Clauses) != 3 {
		t.Fatalf("clauses: got %d want 3", len(sw.Clauses))
	}
	if len(sw.Clauses[0].Values) != 2 {
		t.Fatalf("first case values: got %d want 2", len(sw.Clauses[0].Values))
	}
	if !sw.Clauses[2].IsDefault {
		t.Fatal("expected default clause last")
	}
}

func TestParse_switch_booleanForm(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	switch {
	case true:
		println("yes")
	case false:
		println("no")
	}
}
`
	p := NewTestParser(src, ast.SetupTestLogger(nil))
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	mainFn := nodes[len(nodes)-1].(ast.FunctionNode)
	sw := assertNodeType[*ast.SwitchNode](t, mainFn.Body[0], "*ast.SwitchNode")
	if sw.Tag != nil {
		t.Fatal("boolean switch must have nil tag")
	}
}

func TestParse_switch_fallthrough(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	switch 1 {
	case 1:
		println("one")
		fallthrough
	case 2:
		println("two")
	}
}
`
	p := NewTestParser(src, ast.SetupTestLogger(nil))
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	mainFn := nodes[len(nodes)-1].(ast.FunctionNode)
	sw := assertNodeType[*ast.SwitchNode](t, mainFn.Body[0], "*ast.SwitchNode")
	if len(sw.Clauses[0].Body) != 2 {
		t.Fatalf("case body stmts: got %d want 2", len(sw.Clauses[0].Body))
	}
	if _, ok := sw.Clauses[0].Body[1].(ast.FallthroughNode); !ok {
		t.Fatalf("second stmt: got %T want FallthroughNode", sw.Clauses[0].Body[1])
	}
}

func TestParse_switch_rejectsTypeSwitch(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	switch v := x.(type) {
	case int:
		println("int")
	}
}
`
	p := NewTestParser(src, ast.SetupTestLogger(nil))
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_, _ = p.ParseFile()
	}()
	if recovered == nil {
		t.Fatal("expected parse failure for type switch")
	}
	pe, ok := recovered.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T: %v", recovered, recovered)
	}
	if !strings.Contains(pe.Msg, "type switches") {
		t.Fatalf("unexpected error: %s", pe.Msg)
	}
}
