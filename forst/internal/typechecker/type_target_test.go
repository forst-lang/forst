package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestIsRuntimeEnsureTypeTarget_constrainedScalarAlias(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	str := ast.TypeString
	sku := ast.TypeIdent("Sku")
	tc.registerType(ast.TypeDefNode{
		Ident: sku,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &str,
				Constraints: []ast.ConstraintNode{
					{Name: "Min", Args: []ast.ConstraintArgumentNode{{Value: intVal(1)}}},
					{Name: "Max", Args: []ast.ConstraintArgumentNode{{Value: intVal(64)}}},
				},
			},
		},
	})

	if !tc.isRuntimeEnsureTypeTarget(sku) {
		t.Fatal("expected Sku constrained alias to be a runtime ensure type target")
	}
	carrier, ok := tc.carrierTypeForNamedType(ast.TypeNode{Ident: sku, TypeKind: ast.TypeKindUserDefined})
	if !ok || carrier.Ident != ast.TypeString {
		t.Fatalf("carrier: ok=%v ident=%s", ok, carrier.Ident)
	}
	assertion, ok := tc.ConstrainedScalarAliasAssertion(sku)
	if !ok || assertion == nil || len(assertion.Constraints) != 2 {
		t.Fatalf("ConstrainedScalarAliasAssertion: ok=%v constraints=%v", ok, assertion)
	}
}

func TestIsRuntimeEnsureTypeTarget_bareNominalScalar(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	str := ast.TypeString
	password := ast.TypeIdent("Password")
	tc.registerType(ast.TypeDefNode{
		Ident: password,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{BaseType: &str},
		},
	})
	if !tc.isRuntimeEnsureTypeTarget(password) {
		t.Fatal("expected bare Password = String to be a runtime ensure type target")
	}
	if _, ok := tc.ConstrainedScalarAliasAssertion(password); ok {
		t.Fatal("bare nominal scalar must not report constrained alias assertion")
	}
}

func TestIsRuntimeEnsureTypeTarget_structuralShapeRejected(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	shapeBase := ast.TypeShape
	user := ast.TypeIdent("User")
	tc.registerType(ast.TypeDefNode{
		Ident: user,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &shapeBase,
				Constraints: []ast.ConstraintNode{
					{Name: "Match"},
				},
			},
		},
	})
	if tc.isRuntimeEnsureTypeTarget(user) {
		t.Fatal("structural shape must not be a runtime ensure type target")
	}
	if _, ok := tc.ConstrainedScalarAliasAssertion(user); ok {
		t.Fatal("structural shape must not be a constrained scalar alias")
	}
}

func intVal(n int64) *ast.ValueNode {
	v := ast.ValueNode(ast.IntLiteralNode{Value: n})
	return &v
}
