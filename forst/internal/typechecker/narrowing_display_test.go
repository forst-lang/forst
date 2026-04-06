package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestNarrowingPredicateDisplayFromIsRHS_assertionBaseOnly(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	ms := ast.TypeIdent("MyStr")
	a := ast.AssertionNode{BaseType: &ms}
	got := tc.narrowingPredicateDisplayFromIsRHS(a)
	if got != "MyStr()" {
		t.Fatalf("got %q want MyStr()", got)
	}
}

func TestNarrowingPredicateDisplayFromIsRHS_builtinBaseWithConstraint(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	str := ast.TypeString
	a := ast.AssertionNode{
		BaseType: &str,
		Constraints: []ast.ConstraintNode{
			{Name: "Min", Args: []ast.ConstraintArgumentNode{{Value: ptrVal(ast.IntLiteralNode{Value: 12})}}},
		},
	}
	got := tc.narrowingPredicateDisplayFromIsRHS(a)
	if got != "Min(12)" {
		t.Fatalf("got %q want Min(12)", got)
	}
}

func TestNarrowingPredicateDisplayFromIsRHS_nonBuiltinBaseWithConstraint(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	ms := ast.TypeIdent("MyStr")
	a := ast.AssertionNode{
		BaseType: &ms,
		Constraints: []ast.ConstraintNode{
			{Name: "Min", Args: []ast.ConstraintArgumentNode{{Value: ptrVal(ast.IntLiteralNode{Value: 5})}}},
		},
	}
	got := tc.narrowingPredicateDisplayFromIsRHS(a)
	if got != "MyStr().Min(5)" {
		t.Fatalf("got %q want MyStr().Min(5)", got)
	}
}

func ptrVal(v ast.ValueNode) *ast.ValueNode {
	return &v
}
