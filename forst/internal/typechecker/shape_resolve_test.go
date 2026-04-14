package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestResolveShapeFieldsFromAssertion_nilAndBaseShape(t *testing.T) {
	tc := New(logrus.New(), false)

	got := tc.resolveShapeFieldsFromAssertion(nil)
	if len(got) != 0 {
		t.Fatalf("expected empty map for nil assertion, got %+v", got)
	}

	base := ast.TypeIdent("BaseShape")
	tc.Defs[base] = ast.TypeDefNode{
		Ident: base,
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"id": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}

	got = tc.resolveShapeFieldsFromAssertion(&ast.AssertionNode{BaseType: &base})
	if len(got) != 1 || got["id"].Type == nil || got["id"].Type.Ident != ast.TypeString {
		t.Fatalf("expected merged base shape field id:string, got %+v", got)
	}
}

func TestResolveShapeFieldsFromAssertion_constraintParamMapping(t *testing.T) {
	tc := New(logrus.New(), false)

	guard := ast.TypeGuardNode{
		Ident: "HasCtx",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "s"},
			Type:  ast.TypeNode{Ident: ast.TypeShape},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: "ctx"},
				Type:  ast.TypeNode{Ident: ast.TypeShape},
			},
		},
	}
	tc.Defs[ast.TypeIdent("HasCtx")] = guard

	got := tc.resolveShapeFieldsFromAssertion(&ast.AssertionNode{
		Constraints: []ast.ConstraintNode{
			{
				Name: "HasCtx",
				Args: []ast.ConstraintArgumentNode{
					{Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	})
	field, ok := got["ctx"]
	if !ok || field.Assertion == nil || field.Assertion.BaseType == nil {
		t.Fatalf("expected ctx field assertion mapping, got %+v", got)
	}
	if *field.Assertion.BaseType != ast.TypeString {
		t.Fatalf("expected ctx base type TYPE_STRING, got %q", *field.Assertion.BaseType)
	}
}
