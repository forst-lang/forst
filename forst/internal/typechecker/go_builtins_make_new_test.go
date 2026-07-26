package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func typecheckMakeNewSource(t *testing.T, src string) error {
	t.Helper()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	return tc.CheckTypes(nodes)
}

func TestDispatchMakeNew_sliceMapAndNew(t *testing.T) {
	t.Parallel()
	if err := typecheckMakeNewSource(t, `package main

func main() {
	xs := make(Array(Int), 10)
	xsCap := make(Array(Int), 10, 20)
	m := make(map[String]Int)
	mHint := make(map[String]Int, 4)
	p := new(Int)
	println(len(xs))
	println(len(xsCap))
	println(len(m))
	println(len(mHint))
	if p != nil {
		println("ok")
	}
}
`); err != nil {
		t.Fatal(err)
	}
}

func TestDispatchMakeNew_rejectsNonTypeFirstArg(t *testing.T) {
	t.Parallel()
	fn := BuiltinFunctions["make"]
	_, err := New(logrus.New(), false).checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
		ast.StringLiteralNode{Value: "x"},
		ast.IntLiteralNode{Value: 1},
	}, nil, ast.SourceSpan{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Forst type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatchMakeNew_rejectsWrongMakeArity(t *testing.T) {
	t.Parallel()
	err := typecheckMakeNewSource(t, `package main

func main() {
	xs := make(Array(Int))
	println(len(xs))
}
`)
	if err == nil {
		t.Fatal("expected arity error for make(Array(Int))")
	}
	if !strings.Contains(err.Error(), "2 or 3 arguments") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatchMakeNew_newReturnsPointer(t *testing.T) {
	t.Parallel()
	if err := typecheckMakeNewSource(t, `package main

func main() {
	p := new(Int)
	if p != nil {
		println("ok")
	}
}
`); err != nil {
		t.Fatal(err)
	}
}
