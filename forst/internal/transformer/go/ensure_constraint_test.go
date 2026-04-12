package transformergo

import (
	"errors"
	"strings"
	"testing"

	"forst/internal/ast"
	goast "go/ast"
	gotoken "go/token"
)

func TestTransformConstraintArg_branches(t *testing.T) {
	t.Parallel()
	tn := ast.TypeNode{Ident: ast.TypeString}
	got, err := transformConstraintArg(ast.ConstraintArgumentNode{Type: &tn})
	if err != nil || got.(*goast.Ident).Name != string(tn.Ident) {
		t.Fatalf("type arg: got %v err %v", got, err)
	}

	got, err = transformConstraintArg(ast.ConstraintArgumentNode{Shape: &ast.ShapeNode{}})
	if err != nil || got.(*goast.Ident).Name != "struct{}" {
		t.Fatalf("shape arg: got %v err %v", got, err)
	}

	var v1 ast.ValueNode = ast.IntLiteralNode{Value: 7}
	got, err = transformConstraintArg(ast.ConstraintArgumentNode{Value: &v1})
	if err != nil {
		t.Fatal(err)
	}
	if lit, ok := got.(*goast.BasicLit); !ok || lit.Kind != gotoken.INT || lit.Value != "7" {
		t.Fatalf("int: got %#v", got)
	}

	var v2 ast.ValueNode = ast.StringLiteralNode{Value: "hi"}
	got, err = transformConstraintArg(ast.ConstraintArgumentNode{Value: &v2})
	if err != nil {
		t.Fatal(err)
	}
	if lit, ok := got.(*goast.BasicLit); !ok || lit.Kind != gotoken.STRING || !strings.Contains(lit.Value, "hi") {
		t.Fatalf("string: got %#v", got)
	}

	var v3 ast.ValueNode = ast.BoolLiteralNode{Value: true}
	got, err = transformConstraintArg(ast.ConstraintArgumentNode{Value: &v3})
	if err != nil {
		t.Fatal(err)
	}
	if id, ok := got.(*goast.Ident); !ok || id.Name != "true" {
		t.Fatalf("bool: got %#v", got)
	}

	var v4 ast.ValueNode = ast.FloatLiteralNode{Value: 1.5}
	got, err = transformConstraintArg(ast.ConstraintArgumentNode{Value: &v4})
	if err != nil {
		t.Fatal(err)
	}
	if lit, ok := got.(*goast.BasicLit); !ok || lit.Kind != gotoken.FLOAT {
		t.Fatalf("float: got %#v", got)
	}

	var v5 ast.ValueNode = ast.VariableNode{Ident: ast.Ident{ID: "z"}}
	got, err = transformConstraintArg(ast.ConstraintArgumentNode{Value: &v5})
	if err != nil {
		t.Fatal(err)
	}
	if id, ok := got.(*goast.Ident); !ok || id.Name != "z" {
		t.Fatalf("var: got %#v", got)
	}

	_, err = transformConstraintArg(ast.ConstraintArgumentNode{})
	if !errors.Is(err, errConstraintArgNeedsKind) {
		t.Fatalf("empty: got %v", err)
	}

	var v6 ast.ValueNode = ast.ArrayLiteralNode{}
	_, err = transformConstraintArg(ast.ConstraintArgumentNode{Value: &v6})
	if err == nil || !errors.Is(err, errConstraintArgValueKind) || !strings.Contains(err.Error(), "ArrayLiteralNode") {
		t.Fatalf("unsupported kind: %v", err)
	}
}
