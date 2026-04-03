package ast

import (
	"strings"
	"testing"
)

func TestShapeNode_String_fields(t *testing.T) {
	s := ShapeNode{Fields: map[string]ShapeFieldNode{"a": {Type: &TypeNode{Ident: TypeInt}}}}
	if !strings.Contains(s.String(), "a") || s.Kind() != NodeKindShape {
		t.Fatal(s.String(), s.Kind())
	}
	bt := TypeIdent("User")
	withBase := ShapeNode{BaseType: &bt, Fields: map[string]ShapeFieldNode{}}
	if withBase.BaseType == nil || *withBase.BaseType != "User" {
		t.Fatal(withBase)
	}
}

func TestShapeFieldNode_String_branches(t *testing.T) {
	sfShape := ShapeFieldNode{Shape: &ShapeNode{Fields: map[string]ShapeFieldNode{"n": {Type: &TypeNode{Ident: TypeInt}}}}}
	if !strings.Contains(sfShape.String(), "n") {
		t.Fatal(sfShape.String())
	}
	bt := TypeIdent("B")
	sfAss := ShapeFieldNode{Assertion: &AssertionNode{BaseType: &bt}}
	if !strings.Contains(sfAss.String(), "B") {
		t.Fatal(sfAss.String())
	}
	sfType := ShapeFieldNode{Type: &TypeNode{Ident: TypeString}}
	if !strings.Contains(sfType.String(), "String") {
		t.Fatal(sfType.String())
	}
	empty := ShapeFieldNode{}
	if empty.String() != "?" {
		t.Fatal(empty.String())
	}
}
