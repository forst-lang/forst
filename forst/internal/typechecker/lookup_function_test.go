package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestLookupFunctionReturnType_undefined(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	fn := &ast.FunctionNode{Ident: ast.Ident{ID: "nope"}}
	_, err := tc.LookupFunctionReturnType(fn)
	if err == nil || !strings.Contains(err.Error(), "undefined function") {
		t.Fatalf("got %v", err)
	}
}

func TestLookupAssertionType_baseTypeOnly_userNamed(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	src := `package main

type Label = String

func main() {}
`
	log := setupTestLogger(nil)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	lbl := ast.TypeIdent("Label")
	a := &ast.AssertionNode{BaseType: &lbl, Constraints: nil}
	ty, err := tc.LookupAssertionType(a)
	if err != nil {
		t.Fatal(err)
	}
	if ty == nil || ty.Ident != "Label" {
		t.Fatalf("got %#v", ty)
	}
}
