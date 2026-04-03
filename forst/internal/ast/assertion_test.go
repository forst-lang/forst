package ast

import "testing"

func TestAssertionNode_ToString_nil_base_and_base_only(t *testing.T) {
	u := TypeIdent("U")
	a1 := AssertionNode{Constraints: []ConstraintNode{{Name: "Min"}}}
	if a1.ToString(nil) != "Min()" {
		t.Fatal(a1.ToString(nil))
	}
	a2 := AssertionNode{BaseType: &u}
	if a2.ToString(&u) != "U" {
		t.Fatal(a2.ToString(&u))
	}
}

func TestConstraintArgumentNode_Kind_shape_branch(t *testing.T) {
	arg := ConstraintArgumentNode{Shape: &ShapeNode{Fields: map[string]ShapeFieldNode{}}}
	if arg.Kind() != NodeKindShape {
		t.Fatal(arg.Kind())
	}
}
