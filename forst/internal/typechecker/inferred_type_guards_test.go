package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestTypeGuardConstraintNamesForInferredType_fromTypeNodeAssertion(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	pw := ast.TypeIdent("Password")
	tc.Defs[pw] = ast.TypeDefNode{
		Ident: pw,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{BaseType: &pw},
		},
	}
	g := ast.TypeGuardNode{Ident: ast.Identifier("Strong")}
	tc.Defs[ast.TypeIdent("Strong")] = &g

	tn := ast.TypeNode{
		Ident: ast.TypeAssertion,
		Assertion: &ast.AssertionNode{
			BaseType: &pw,
			Constraints: []ast.ConstraintNode{
				{Name: "Strong"},
			},
		},
	}
	got := tc.TypeGuardConstraintNamesForInferredType(tn)
	if len(got) != 1 || got[0] != "Strong" {
		t.Fatalf("got %v", got)
	}
}

func TestTypeGuardConstraintNamesForInferredType_skipsNonGuards(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	tn := ast.TypeNode{
		Ident: ast.TypeAssertion,
		Assertion: &ast.AssertionNode{
			Constraints: []ast.ConstraintNode{
				{Name: ast.ValueConstraint},
				{Name: "Min"},
			},
		},
	}
	got := tc.TypeGuardConstraintNamesForInferredType(tn)
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestTypeGuardNamesFromAssertionNode_includesBuiltinConstraints(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	a := &ast.AssertionNode{
		Constraints: []ast.ConstraintNode{
			{Name: "Min"},
			{Name: "Equals"},
		},
	}
	got := tc.typeGuardNamesFromAssertionNode(a)
	if len(got) != 2 || got[0] != "Min" || got[1] != "Equals" {
		t.Fatalf("got %v", got)
	}
}

func TestTypeGuardNamesFromIsRHS_functionCallIncludesNameWithoutDefs(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	call := ast.FunctionCallNode{Function: ast.Ident{ID: "Min"}}
	got := tc.typeGuardNamesFromIsRHS(call)
	if len(got) != 1 || got[0] != "Min" {
		t.Fatalf("got %v", got)
	}
}
