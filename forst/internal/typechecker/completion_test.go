package typechecker

import (
	"slices"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestListFieldNamesForType_builtinString(t *testing.T) {
	log := logrus.New()
	tc := New(log, false)
	names := tc.ListFieldNamesForType(ast.TypeNode{Ident: ast.TypeString})
	if len(names) != 1 || names[0] != "len" {
		t.Fatalf("got %#v", names)
	}
}

func TestListFieldNamesForType_resultOkErr(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	rt := ast.TypeNode{
		Ident: ast.TypeResult,
		TypeParams: []ast.TypeNode{
			{Ident: ast.TypeInt},
			{Ident: ast.TypeError},
		},
	}
	names := tc.ListFieldNamesForType(rt)
	slices.Sort(names)
	if !slices.Equal(names, []string{"Err", "Ok"}) {
		t.Fatalf("got %#v", names)
	}
}

func TestListFieldNamesForType_pointerUnwrapsToBuiltin(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	ptr := ast.TypeNode{
		Ident:      ast.TypePointer,
		TypeParams: []ast.TypeNode{{Ident: ast.TypeString}},
	}
	names := tc.ListFieldNamesForType(ptr)
	if len(names) != 1 || names[0] != "len" {
		t.Fatalf("got %#v", names)
	}
}

func TestListFieldNamesForType_shapeTypeDefFields(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

type Point = { x: Int, y: Int }

func main() {}
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
	names := tc.ListFieldNamesForType(ast.TypeNode{Ident: "Point"})
	slices.Sort(names)
	if !slices.Equal(names, []string{"x", "y"}) {
		t.Fatalf("got %#v", names)
	}
}

func TestListFieldNamesForType_missingDefReturnsEmptySorted(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	names := tc.ListFieldNamesForType(ast.TypeNode{Ident: "NoSuchType"})
	if len(names) != 0 {
		t.Fatalf("got %#v", names)
	}
}

func TestVisibleVariableLikeSymbols_afterRestoreScope_functionParamsAndLocals(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func foo(a: Int) {
	x := 1
	println(x)
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
	var fooFn *ast.FunctionNode
	for _, n := range nodes {
		if fn, ok := n.(ast.FunctionNode); ok && fn.Ident.ID == "foo" {
			fooFn = &fn
			break
		}
	}
	if fooFn == nil {
		t.Fatal("foo not found")
	}
	if err := tc.RestoreScope(fooFn); err != nil {
		t.Fatal(err)
	}
	ids := tc.VisibleVariableLikeSymbols()
	var names []string
	for _, id := range ids {
		names = append(names, string(id))
	}
	slices.Sort(names)
	if !slices.Contains(names, "a") || !slices.Contains(names, "x") {
		t.Fatalf("expected a and x in %#v", names)
	}
}

func TestInferExpressionTypeForIdentifier_matchesInfer(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func main() {
	y := 2
	println(y)
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
	var mainFn *ast.FunctionNode
	for _, n := range nodes {
		if fn, ok := n.(ast.FunctionNode); ok && fn.Ident.ID == "main" {
			mainFn = &fn
			break
		}
	}
	if err := tc.RestoreScope(mainFn); err != nil {
		t.Fatal(err)
	}
	types, err := tc.InferExpressionTypeForCompletion(ast.VariableNode{Ident: ast.Ident{ID: "y"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || types[0].Ident != ast.TypeInt {
		t.Fatalf("got %#v", types)
	}
}
