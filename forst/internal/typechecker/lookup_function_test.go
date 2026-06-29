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

func TestLookupFunctionReturnType_endToEndReceiverMethod(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

type Counter = { n: Int }

func (Counter) inc(): Int {
	return 1
}

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
	var fn *ast.FunctionNode
	for _, n := range nodes {
		if f, ok := n.(ast.FunctionNode); ok && f.Receiver != nil {
			fn = &f
			break
		}
	}
	if fn == nil {
		t.Fatal("no receiver method found")
	}
	rts, err := tc.LookupFunctionReturnType(fn)
	if err != nil || len(rts) != 1 || rts[0].Ident != ast.TypeInt {
		t.Fatalf("return types = %#v err=%v", rts, err)
	}
}

func TestLookupFunctionReturnType_undefinedReceiverMethod(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	counter := ast.TypeIdent("Counter")
	_ = counter
	fn := &ast.FunctionNode{
		Ident: ast.Ident{ID: "missing"},
		Receiver: &ast.SimpleParamNode{
			Type: ast.TypeNode{Ident: "Counter"},
		},
	}
	_, err := tc.LookupFunctionReturnType(fn)
	if err == nil || !strings.Contains(err.Error(), "undefined receiver method") {
		t.Fatalf("got %v", err)
	}
}

func TestLookupAssertionType_constraintAssertion(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	str := ast.TypeString
	a := &ast.AssertionNode{
		BaseType: &str,
		Constraints: []ast.ConstraintNode{{
			Name: "Min",
			Args: []ast.ConstraintArgumentNode{{
				Value: func() *ast.ValueNode {
					v := ast.IntLiteralNode{Value: 1}
					var n ast.ValueNode = v
					return &n
				}(),
			}},
		}},
	}
	ty, err := tc.LookupAssertionType(a)
	if err != nil {
		t.Fatal(err)
	}
	if ty == nil || !ty.IsHashBased() {
		t.Fatalf("expected hash-based assertion type, got %#v", ty)
	}
}
