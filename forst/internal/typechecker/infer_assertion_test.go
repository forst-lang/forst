package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestInferAssertionType_baseTypeNotFound(t *testing.T) {
	tc := testTC(t)
	missing := ast.TypeIdent("NoSuchBase")
	assertion := &ast.AssertionNode{
		BaseType: &missing,
	}

	_, err := tc.InferAssertionType(assertion, false, "", nil)
	if err == nil || !strings.Contains(err.Error(), "NoSuchBase") || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected base type not found error, got %v", err)
	}
}

func TestInferAssertionType_valueConstraintIntLiteral(t *testing.T) {
	tc := testTC(t)
	v := ast.IntLiteralNode{Value: 42}
	var vn ast.ValueNode = v
	assertion := &ast.AssertionNode{
		Constraints: []ast.ConstraintNode{{
			Name: ast.ValueConstraint,
			Args: []ast.ConstraintArgumentNode{{Value: &vn}},
		}},
	}

	types, err := tc.InferAssertionType(assertion, false, "n", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || types[0].Ident != ast.TypeInt {
		t.Fatalf("want Int, got %+v", types)
	}
}

func TestInferAssertionType_valueConstraintStringLiteral(t *testing.T) {
	tc := testTC(t)
	v := ast.StringLiteralNode{Value: "hi"}
	var vn ast.ValueNode = v
	assertion := &ast.AssertionNode{
		Constraints: []ast.ConstraintNode{{
			Name: ast.ValueConstraint,
			Args: []ast.ConstraintArgumentNode{{Value: &vn}},
		}},
	}

	types, err := tc.InferAssertionType(assertion, false, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || types[0].Ident != ast.TypeString {
		t.Fatalf("want String, got %+v", types)
	}
}

func TestInferAssertionType_valueConstraintEmptyArgs(t *testing.T) {
	tc := testTC(t)
	assertion := &ast.AssertionNode{
		Constraints: []ast.ConstraintNode{{
			Name: ast.ValueConstraint,
			Args: nil,
		}},
	}

	_, err := tc.InferAssertionType(assertion, false, "fieldX", nil)
	if err == nil || !strings.Contains(err.Error(), "fieldX") {
		t.Fatalf("expected error for empty Value args, got %v", err)
	}
}

func TestInferAssertionType_baseTypeFieldsMergedBeforeMatch(t *testing.T) {
	tc := testTC(t)
	tc.Defs["Person"] = ast.MakeTypeDef("Person", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"name": ast.MakeTypeField(ast.TypeString),
	}))

	base := ast.TypeIdent("Person")
	assertion := &ast.AssertionNode{
		BaseType: &base,
		Constraints: []ast.ConstraintNode{{
			Name: ConstraintMatch,
			Args: []ast.ConstraintArgumentNode{{
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"age": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
					},
				},
			}},
		}},
	}

	types, err := tc.InferAssertionType(assertion, false, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 {
		t.Fatalf("expected one inferred type, got %+v", types)
	}
	hashIdent := types[0].Ident
	def, ok := tc.Defs[hashIdent].(ast.TypeDefNode)
	if !ok {
		t.Fatalf("expected hash-based type %q registered as TypeDefNode", hashIdent)
	}
	shapeExpr, ok := def.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("expected shape expr, got %T", def.Expr)
	}
	if _, ok := shapeExpr.Shape.Fields["name"]; !ok {
		t.Fatalf("expected base Person field name merged, fields=%v", shapeExpr.Shape.Fields)
	}
	if _, ok := shapeExpr.Shape.Fields["age"]; !ok {
		t.Fatalf("expected Match field age merged, fields=%v", shapeExpr.Shape.Fields)
	}
}
