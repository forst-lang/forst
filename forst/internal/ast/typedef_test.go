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

func TestTypeDefExpr_marker_methods(t *testing.T) {
	TypeDefAssertionExpr{}.isTypeDefExpr()
	TypeDefBinaryExpr{}.isTypeDefExpr()
	TypeDefShapeExpr{}.isTypeDefExpr()
	TypeDefErrorExpr{}.isTypeDefExpr()
}

func TestTypeDefErrorExpr_Kind_and_PayloadShape(t *testing.T) {
	ee := TypeDefErrorExpr{Payload: ShapeNode{Fields: map[string]ShapeFieldNode{}}}
	if ee.Kind() != NodeKindTypeDefError {
		t.Fatalf("Kind: %v", ee.Kind())
	}
	sh, ok := PayloadShape(ee)
	if !ok || sh == nil || len(sh.Fields) != 0 {
		t.Fatalf("PayloadShape(TypeDefErrorExpr): ok=%v fields=%d", ok, len(sh.Fields))
	}
	se := TypeDefShapeExpr{Shape: ShapeNode{Fields: map[string]ShapeFieldNode{}}}
	sh2, ok := PayloadShape(se)
	if !ok || sh2 == nil {
		t.Fatal("PayloadShape(TypeDefShapeExpr)")
	}
}

func TestTypeDefBinaryExpr_and_TypeDefShapeExpr_Kind(t *testing.T) {
	bin := TypeDefBinaryExpr{
		Op:    TokenBitwiseAnd,
		Left:  TypeDefShapeExpr{Shape: ShapeNode{}},
		Right: TypeDefShapeExpr{Shape: ShapeNode{}},
	}
	if bin.Kind() != NodeKindTypeDefBinaryExpr {
		t.Fatal(bin.Kind())
	}
	shapeExpr := TypeDefShapeExpr{Shape: ShapeNode{}}
	if shapeExpr.Kind() != NodeKindTypeDefShape {
		t.Fatal(shapeExpr.Kind())
	}
}
