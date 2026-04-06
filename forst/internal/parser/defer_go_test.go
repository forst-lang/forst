package parser

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestParse_defer_and_go_statements(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	p := NewTestParser(`package main

func f() {}

func main() {
	defer f()
	go f()
}
`, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(nodes) < 2 {
		t.Fatalf("expected at least package + func f + func main, got %d top-level nodes", len(nodes))
	}
	mainFn, ok := nodes[len(nodes)-1].(ast.FunctionNode)
	if !ok {
		t.Fatalf("last decl: got %T", nodes[len(nodes)-1])
	}
	if len(mainFn.Body) != 2 {
		t.Fatalf("main body: got %d stmts", len(mainFn.Body))
	}
	d0 := assertNodeType[*ast.DeferNode](t, mainFn.Body[0], "*ast.DeferNode")
	if _, ok := d0.Call.(ast.FunctionCallNode); !ok {
		t.Fatalf("defer call: got %T", d0.Call)
	}
	g1 := assertNodeType[*ast.GoStmtNode](t, mainFn.Body[1], "*ast.GoStmtNode")
	if _, ok := g1.Call.(ast.FunctionCallNode); !ok {
		t.Fatalf("go call: got %T", g1.Call)
	}
}

func TestParse_defer_rejects_parenthesized_operand(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	src := `package main

func f() {}

func main() {
	defer (f())
}
`
	p := NewTestParser(src, log)
	var recovered interface{}
	func() {
		defer func() { recovered = recover() }()
		_, _ = p.ParseFile()
	}()
	if recovered == nil {
		t.Fatal("expected panic from FailWithParseError")
	}
	pe, ok := recovered.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T: %v", recovered, recovered)
	}
	if !strings.Contains(pe.Msg, "parenthesized") {
		t.Fatalf("unexpected message: %s", pe.Msg)
	}
}

func TestParse_go_rejects_parenthesized_operand(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	src := `package main

func f() {}

func main() {
	go (f())
}
`
	p := NewTestParser(src, log)
	var recovered interface{}
	func() {
		defer func() { recovered = recover() }()
		_, _ = p.ParseFile()
	}()
	if recovered == nil {
		t.Fatal("expected panic from FailWithParseError")
	}
	pe, ok := recovered.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T: %v", recovered, recovered)
	}
	if !strings.Contains(pe.Msg, "parenthesized") {
		t.Fatalf("unexpected message: %s", pe.Msg)
	}
}
