package transformergo

import (
	goast "go/ast"
	"testing"

	"forst/internal/ast"
)

func TestTransformTypeDefExpr_valueConstraintLiteralKinds(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	testCases := []struct {
		name     string
		value    ast.ValueNode
		expected string
	}{
		{name: "string literal", value: ast.StringLiteralNode{Value: "x"}, expected: "string"},
		{name: "int literal", value: ast.IntLiteralNode{Value: 1}, expected: "int"},
		{name: "float literal", value: ast.FloatLiteralNode{Value: 1.5}, expected: "float64"},
		{name: "bool literal", value: ast.BoolLiteralNode{Value: true}, expected: "bool"},
		{name: "variable value", value: ast.VariableNode{Ident: ast.Ident{ID: "v"}}, expected: "string"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			v := tc.value
			expr, err := tr.transformTypeDefExpr(ast.TypeDefAssertionExpr{
				Assertion: &ast.AssertionNode{
					Constraints: []ast.ConstraintNode{{
						Name: ast.ValueConstraint,
						Args: []ast.ConstraintArgumentNode{{Value: &v}},
					}},
				},
			})
			if err != nil {
				t.Fatalf("transformTypeDefExpr: %v", err)
			}
			ident, ok := (*expr).(*goast.Ident)
			if !ok || ident.Name != tc.expected {
				t.Fatalf("expected %q ident, got %#v", tc.expected, *expr)
			}
		})
	}
}

func TestTransformTypeDefExpr_builtinShapeAndErrorBranches(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	base := ast.TypeIdent(ast.TypeInt)
	builtinExpr, err := tr.transformTypeDefExpr(ast.TypeDefAssertionExpr{
		Assertion: &ast.AssertionNode{BaseType: &base},
	})
	if err != nil {
		t.Fatalf("builtin transformTypeDefExpr: %v", err)
	}
	if id, ok := (*builtinExpr).(*goast.Ident); !ok || id.Name != "int" {
		t.Fatalf("expected built-in int ident, got %#v", *builtinExpr)
	}

	shapeExpr, err := tr.transformTypeDefExpr(ast.TypeDefShapeExpr{
		Shape: ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			},
		},
	})
	if err != nil || shapeExpr == nil {
		t.Fatalf("shape transformTypeDefExpr failed: %v", err)
	}

	errorExpr, err := tr.transformTypeDefExpr(ast.TypeDefErrorExpr{
		Payload: ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"message": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			},
		},
	})
	if err != nil || errorExpr == nil {
		t.Fatalf("error-payload transformTypeDefExpr failed: %v", err)
	}
}

func TestDefineShapeFields_registersNestedShapeTypes(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	shape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"meta": {
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"id": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
					},
				},
			},
		},
	}
	if err := tr.defineShapeFields(shape); err != nil {
		t.Fatalf("defineShapeFields: %v", err)
	}
	if len(tr.Output.types) == 0 {
		t.Fatal("expected nested shape type declarations to be emitted")
	}
}

func TestTransformTypeDefExpr_pointerAssertionExpr_deref(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	base := ast.TypeIdent(ast.TypeInt)
	inner := ast.TypeDefAssertionExpr{
		Assertion: &ast.AssertionNode{BaseType: &base},
	}
	expr, err := tr.transformTypeDefExpr(&inner)
	if err != nil {
		t.Fatal(err)
	}
	if expr == nil || *expr == nil {
		t.Fatal("expected Go expr")
	}
}

