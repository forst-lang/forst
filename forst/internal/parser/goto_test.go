package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseGoto_basic(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	goto done
	println(1)
done:
	println(2)
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	mainFn := nodes[1].(ast.FunctionNode)
	g, ok := mainFn.Body[0].(*ast.GotoNode)
	if !ok || g.Label == nil || g.Label.ID != "done" {
		t.Fatalf("body[0] = %#v, want GotoNode done", mainFn.Body[0])
	}
	labeled, ok := mainFn.Body[2].(*ast.LabeledStmtNode)
	if !ok || labeled.Label == nil || labeled.Label.ID != "done" {
		t.Fatalf("body[2] = %#v, want LabeledStmtNode done", mainFn.Body[2])
	}
}

func TestParseGoto_missingLabel(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	goto
}
`
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_, _ = NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	}()
	if recovered == nil {
		t.Fatal("expected parse error for goto without label")
	}
}

func TestParseLabeled_forLoop(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
outer:
	for {
		break outer
	}
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	mainFn := nodes[1].(ast.FunctionNode)
	fn, ok := mainFn.Body[0].(*ast.ForNode)
	if !ok || fn.Label == nil || fn.Label.ID != "outer" {
		t.Fatalf("body[0] = %#v, want labeled ForNode", mainFn.Body[0])
	}
}

func TestParseLabeled_vsTypedAssign(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	x: Int = 1
	println(x)
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	mainFn := nodes[1].(ast.FunctionNode)
	as, ok := mainFn.Body[0].(ast.AssignmentNode)
	if !ok {
		t.Fatalf("body[0] = %T, want AssignmentNode", mainFn.Body[0])
	}
	if !as.IsShort && (len(as.ExplicitTypes) == 0 || as.ExplicitTypes[0] == nil) {
		// typed form uses ExplicitTypes; IsShort is for :=
	}
	if len(as.ExplicitTypes) == 0 || as.ExplicitTypes[0] == nil || as.ExplicitTypes[0].Ident != ast.TypeInt {
		t.Fatalf("expected typed Int assignment, got %#v", as)
	}
}
