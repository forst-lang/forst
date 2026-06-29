package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func TestGetLeftmostVariable_nestedBinary(t *testing.T) {
	t.Parallel()
	tc := New(setupTestLogger(nil), false)
	left := ast.BinaryExpressionNode{
		Left:  ast.VariableNode{Ident: ast.Ident{ID: "a"}},
		Right: ast.IntLiteralNode{Value: 1},
	}
	got, err := tc.getLeftmostVariable(left)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := got.(ast.VariableNode); !ok || v.Ident.ID != "a" {
		t.Fatalf("got %#v", got)
	}
}

func TestGetLeftmostVariable_nonVariableErrors(t *testing.T) {
	t.Parallel()
	tc := New(setupTestLogger(nil), false)
	_, err := tc.getLeftmostVariable(ast.IntLiteralNode{Value: 1})
	if err == nil {
		t.Fatal("expected error for literal LHS")
	}
}

func TestUnifyIsOperator_shapeLiteral(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main
type Row = { id: Int, name: String }
func main() {
	r := Row{ id: 1, name: "a" }
	if r is { id: 1 } {
		println("ok")
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
		t.Fatal(err)
	}
}

func TestUnifyIsOperator_resultOkGuard(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main
func f(): Result(Int, Error) {
	return 1
}
func main() {
	x := f()
	if x is Ok() {
		println("ok")
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
		t.Fatal(err)
	}
}
