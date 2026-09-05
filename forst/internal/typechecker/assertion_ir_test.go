package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestLowerAssertionNode_orToAny(t *testing.T) {
	a := ast.AssertionNode{
		Constraints: []ast.ConstraintNode{{Name: "A"}},
		OrChains: []ast.AssertionNode{
			{Constraints: []ast.ConstraintNode{{Name: "B"}}},
		},
	}
	ir := LowerAssertionNode(a)
	shape := AssertionShape(ir)
	if shape != "Any(Atom(A),Atom(B))" {
		t.Fatalf("shape=%s", shape)
	}
}

func TestLowerAssertionNode_meetToAll(t *testing.T) {
	a := ast.AssertionNode{
		Constraints: []ast.ConstraintNode{{Name: "Min"}, {Name: "Max"}},
	}
	ir := LowerAssertionNode(a)
	if AssertionShape(ir) != "All(Atom(Min),Atom(Max))" {
		t.Fatalf("shape=%s", AssertionShape(ir))
	}
}

func TestLowerTypeGuardBody_sequentialAll(t *testing.T) {
	body := []ast.Node{
		ast.EnsureNode{Assertion: ast.AssertionNode{Constraints: []ast.ConstraintNode{{Name: "A"}}}},
		ast.EnsureNode{Assertion: ast.AssertionNode{Constraints: []ast.ConstraintNode{{Name: "B"}}}},
	}
	ir := LowerTypeGuardBody(body)
	if AssertionShape(ir) != "All(Atom(A),Atom(B))" {
		t.Fatalf("shape=%s", AssertionShape(ir))
	}
}

func TestLowerTypeGuardBody_noDNF(t *testing.T) {
	body := []ast.Node{
		ast.EnsureNode{Assertion: ast.AssertionNode{
			Constraints: []ast.ConstraintNode{{Name: "A"}},
			OrChains:    []ast.AssertionNode{{Constraints: []ast.ConstraintNode{{Name: "B"}}}},
		}},
		ast.EnsureNode{Assertion: ast.AssertionNode{Constraints: []ast.ConstraintNode{{Name: "C"}}}},
	}
	ir := LowerTypeGuardBody(body)
	shape := AssertionShape(ir)
	if shape != "All(Any(Atom(A),Atom(B)),Atom(C))" {
		t.Fatalf("want nested All(Any,C), got %s", shape)
	}
	if strings.HasPrefix(shape, "Any(All(") {
		t.Fatalf("DNF expansion forbidden: %s", shape)
	}
}

func TestLowerRefinementTarget_typeTargetNotAtom(t *testing.T) {
	tt := ast.TypeTarget{Name: "ActiveStatus"}
	base := ast.TypeIdent("ActiveStatus")
	a, got := LowerRefinementTarget(tt, ast.AssertionNode{BaseType: &base})
	if a != nil {
		t.Fatalf("TypeTarget must not lower to assertion, got %s", AssertionShape(a))
	}
	if got == nil || got.Name != "ActiveStatus" {
		t.Fatalf("TypeTarget: %#v", got)
	}
}

func TestLowerAssertion_runtimeMin(t *testing.T) {
	var n ast.ValueNode = ast.VariableNode{Ident: ast.Ident{ID: "n"}}
	a := ast.AssertionNode{
		Constraints: []ast.ConstraintNode{{
			Name: "Min",
			Args: []ast.ConstraintArgumentNode{{Value: &n}},
		}},
	}
	ir := LowerAssertionNode(a)
	atom, ok := ir.(Atom)
	if !ok || !atom.RuntimeOnly {
		t.Fatalf("want runtime Atom, got %#v", ir)
	}
	if !HasRuntimeOnlyAtom(ir) {
		t.Fatal("HasRuntimeOnlyAtom")
	}
}
