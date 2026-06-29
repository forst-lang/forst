package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func TestInferAssignmentTypes_shortDeclInfersTypes(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func main() {
	x := 1
	s := "hello"
	println(x)
	println(s)
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
	xTy, ok := tc.VariableTypes[ast.Identifier("x")]
	if !ok || len(xTy) != 1 || xTy[0].Ident != ast.TypeInt {
		t.Fatalf("x types = %#v ok=%v", xTy, ok)
	}
	sTy, ok := tc.VariableTypes[ast.Identifier("s")]
	if !ok || len(sTy) != 1 || sTy[0].Ident != ast.TypeString {
		t.Fatalf("s types = %#v ok=%v", sTy, ok)
	}
}

func TestInferAssignmentTypes_rejectsExplicitTypesOnCompound(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	chk := New(log, false)
	chk.VariableTypes[ast.Identifier("n")] = []ast.TypeNode{{Ident: ast.TypeInt}}
	assign := ast.AssignmentNode{
		LValues:       []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "n"}}},
		RValues:       []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
		ExplicitTypes: []*ast.TypeNode{{Ident: ast.TypeInt}},
		CompoundOp:    ast.TokenPlusEq,
	}
	if err := chk.inferAssignmentTypes(assign); err == nil {
		t.Fatal("expected error for explicit types on compound assign")
	}
}

func TestInferAssignmentTypes_rejectsMultiLValueCompound(t *testing.T) {
	t.Parallel()
	chk := New(setupTestLogger(nil), false)
	assign := ast.AssignmentNode{
		LValues: []ast.ExpressionNode{
			ast.VariableNode{Ident: ast.Ident{ID: "a"}},
			ast.VariableNode{Ident: ast.Ident{ID: "b"}},
		},
		RValues:    []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
		CompoundOp: ast.TokenPlusEq,
	}
	if err := chk.inferAssignmentTypes(assign); err == nil {
		t.Fatal("expected error for multi-value compound assign")
	}
}

func TestInferAssignmentTypes_regularAssign(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func main() {
	var x Int
	x = 42
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
	ty, ok := tc.VariableTypes[ast.Identifier("x")]
	if !ok || len(ty) != 1 || ty[0].Ident != ast.TypeInt {
		t.Fatalf("x types = %#v ok=%v", ty, ok)
	}
}
