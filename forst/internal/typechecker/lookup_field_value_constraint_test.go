package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestInferValueConstraintType_basicErrors(t *testing.T) {
	tc := New(logrus.New(), false)

	_, err := tc.inferValueConstraintType(ast.ConstraintNode{Name: ast.ValueConstraint}, "f", nil)
	if err == nil {
		t.Fatal("expected error for missing Value args")
	}

	c := ast.ConstraintNode{
		Name: ast.ValueConstraint,
		Args: []ast.ConstraintArgumentNode{{Value: nil}},
	}
	_, err = tc.inferValueConstraintType(c, "f", nil)
	if err == nil {
		t.Fatal("expected error for nil Value arg")
	}
}

func TestInferValueConstraintType_literalAndFallbackExpectedType(t *testing.T) {
	tc := New(logrus.New(), false)

	vs := ast.ValueNode(ast.StringLiteralNode{Value: "abc"})
	typ, err := tc.inferValueConstraintType(ast.ConstraintNode{
		Name: ast.ValueConstraint,
		Args: []ast.ConstraintArgumentNode{{Value: &vs}},
	}, "name", nil)
	if err != nil {
		t.Fatalf("string literal infer: %v", err)
	}
	if typ.Ident != ast.TypeString {
		t.Fatalf("expected TYPE_STRING, got %q", typ.Ident)
	}

	vVar := ast.ValueNode(ast.VariableNode{Ident: ast.Ident{ID: "missing"}})
	expected := ast.TypeNode{Ident: ast.TypeBool}
	typ, err = tc.inferValueConstraintType(ast.ConstraintNode{
		Name: ast.ValueConstraint,
		Args: []ast.ConstraintArgumentNode{{Value: &vVar}},
	}, "flag", &expected)
	if err != nil {
		t.Fatalf("fallback expected type infer: %v", err)
	}
	if typ.Ident != ast.TypeBool {
		t.Fatalf("expected fallback TYPE_BOOL, got %q", typ.Ident)
	}
}
