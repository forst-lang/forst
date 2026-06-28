package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func strLit(v string) *ast.ValueNode {
	n := ast.ValueNode(ast.StringLiteralNode{Value: v})
	return &n
}

func TestInferAssertion_minConstraintOnStringParam(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := New(log, false)
	src := `package main

func check(name String) {
	ensure name is Min(1)
}

func main() {
	check("a")
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("Min constraint pipeline: %v", err)
	}
}

func TestInferAssertion_matchConstraintMergesFields(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := New(log, false)
	base := ast.TypeIdent("Row")
	tc.Defs["Row"] = ast.TypeDefNode{
		Ident: base,
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"id": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	}
	assertion := &ast.AssertionNode{
		BaseType: &base,
		Constraints: []ast.ConstraintNode{{
			Name: "Match",
			Args: []ast.ConstraintArgumentNode{{
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
					},
				},
			}},
		}},
	}
	types, err := tc.InferAssertionType(assertion, false, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) == 0 {
		t.Fatal("expected inferred types")
	}
	def, ok := tc.Defs[types[0].Ident].(ast.TypeDefNode)
	if !ok {
		t.Fatalf("def type %T", tc.Defs[types[0].Ident])
	}
	shape, ok := def.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("expr type %T", def.Expr)
	}
	if _, ok := shape.Shape.Fields["id"]; !ok {
		t.Fatal("missing id from base")
	}
	if _, ok := shape.Shape.Fields["name"]; !ok {
		t.Fatal("missing name from Match")
	}
}

func TestInferAssertion_valueConstraintLiteral(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := New(log, false)
	assertion := &ast.AssertionNode{
		Constraints: []ast.ConstraintNode{{
			Name: ast.ValueConstraint,
			Args: []ast.ConstraintArgumentNode{{
				Value: strLit("admin"),
			}},
		}},
	}
	types, err := tc.InferAssertionType(assertion, false, "role", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || types[0].Ident != ast.TypeString {
		t.Fatalf("got %#v", types)
	}
}

func TestInferAssertion_unknownBaseTypeErrors(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := New(log, false)
	assertion := &ast.AssertionNode{
		BaseType: ptrTypeIdent(ast.TypeIdent("NoSuchType")),
	}
	if _, err := tc.InferAssertionType(assertion, false, "x", nil); err == nil {
		t.Fatal("expected unknown base type error")
	}
}
