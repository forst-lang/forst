package ast

import (
	"strings"
	"testing"
)

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

func TestConstraintNode_String_empty_args(t *testing.T) {
	c := ConstraintNode{Name: "Min", Args: []ConstraintArgumentNode{}}
	if c.String() != "Min()" {
		t.Fatal(c.String())
	}
}

func TestConstraintNode_String_multiple_args(t *testing.T) {
	c := ConstraintNode{
		Name: "Between",
		Args: []ConstraintArgumentNode{
			{Type: &TypeNode{Ident: TypeInt}},
			{Type: &TypeNode{Ident: TypeString}},
		},
	}
	s := c.String()
	if !strings.Contains(s, "Between") || !strings.Contains(s, "Int") || !strings.Contains(s, "String") || !strings.Contains(s, ",") {
		t.Fatal(s)
	}
}

func TestConstraintArgumentNode_Kind_value_path_and_String_type_only(t *testing.T) {
	vn := ValueNode(VariableNode{Ident: Ident{ID: "a"}})
	arg := ConstraintArgumentNode{Value: &vn}
	if arg.Kind() != NodeKindVariable {
		t.Fatal(arg.Kind())
	}
	arg2 := ConstraintArgumentNode{Type: &TypeNode{Ident: TypeInt}}
	if arg2.String() != "Int" {
		t.Fatal(arg2.String())
	}
}

func TestAssertionNode_ToString_base_and_constraints_together(t *testing.T) {
	base := TypeIdent("S")
	a := AssertionNode{
		BaseType:    &base,
		Constraints: []ConstraintNode{{Name: "Min", Args: []ConstraintArgumentNode{}}},
	}
	s := a.ToString(&base)
	if !strings.Contains(s, "S") || !strings.Contains(s, "Min()") {
		t.Fatal(s)
	}
}

func TestAssertionNode_isExpression_marker(t *testing.T) {
	AssertionNode{}.isExpression()
}
