package ast

import (
	"strings"
	"testing"
)

func TestTypeDefAssertionExpr_String_and_Kind(t *testing.T) {
	aNil := TypeDefAssertionExpr{}
	if aNil.String() != "TypeDefAssertionExpr(?)" || aNil.Kind() != NodeKindTypeDefAssertion {
		t.Fatal(aNil.String(), aNil.Kind())
	}
	base := TypeIdent("U")
	a := TypeDefAssertionExpr{Assertion: &AssertionNode{BaseType: &base}}
	if !strings.Contains(a.String(), "U") {
		t.Fatal(a.String())
	}
}

func TestTypeDefBinaryExpr_conjunction_disjunction_String(t *testing.T) {
	and := TypeDefBinaryExpr{
		Op:    TokenBitwiseAnd,
		Left:  TypeDefShapeExpr{Shape: ShapeNode{Fields: map[string]ShapeFieldNode{}}},
		Right: TypeDefShapeExpr{Shape: ShapeNode{Fields: map[string]ShapeFieldNode{}}},
	}
	if !and.IsConjunction() || and.IsDisjunction() {
		t.Fatal("conjunction")
	}
	or := TypeDefBinaryExpr{
		Op:    TokenBitwiseOr,
		Left:  TypeDefShapeExpr{Shape: ShapeNode{}},
		Right: TypeDefShapeExpr{Shape: ShapeNode{}},
	}
	if !or.IsDisjunction() || or.IsConjunction() {
		t.Fatal("disjunction")
	}
	if !strings.Contains(and.String(), "TypeDefBinaryExpr") {
		t.Fatal(and.String())
	}
}
