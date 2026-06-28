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

func TestShapeNode_String_unknown_field_cell(t *testing.T) {
	s := ShapeNode{Fields: map[string]ShapeFieldNode{"x": {}}}
	if !strings.Contains(s.String(), "?") {
		t.Fatal(s.String())
	}
}

func TestShapeNode_isExpression_marker(_ *testing.T) {
	ShapeNode{}.isExpression()
}

func TestShapeNode_String_field_nested_shape_branch(t *testing.T) {
	inner := ShapeNode{Fields: map[string]ShapeFieldNode{
		"n": {Type: &TypeNode{Ident: TypeInt}},
	}}
	s := ShapeNode{Fields: map[string]ShapeFieldNode{
		"outer": {Shape: &inner},
	}}
	out := s.String()
	if !strings.Contains(out, "outer") || !strings.Contains(out, "n") {
		t.Fatal(out)
	}
}

func TestShapeNode_String_empty_fields_map(t *testing.T) {
	if got := (ShapeNode{Fields: map[string]ShapeFieldNode{}}).String(); got != "{}" {
		t.Fatal(got)
	}
}

func TestShapeNode_String_multiple_top_level_fields(t *testing.T) {
	s := ShapeNode{Fields: map[string]ShapeFieldNode{
		"a": {Type: &TypeNode{Ident: TypeInt}},
		"b": {Type: &TypeNode{Ident: TypeString}},
	}}
	out := s.String()
	if !strings.Contains(out, "a:") || !strings.Contains(out, "b:") || !strings.Contains(out, ", ") {
		t.Fatal(out)
	}
}

func TestShapeFieldNode_IsMethodField_and_methodString(t *testing.T) {
	field := ShapeFieldNode{
		IsMethod: true,
		MethodParams: []ParamNode{
			SimpleParamNode{Ident: Ident{ID: "msg"}, Type: TypeNode{Ident: TypeString}},
		},
		MethodReturnTypes: []TypeNode{{Ident: TypeError}},
	}
	if !field.IsMethodField() {
		t.Fatal("expected method field")
	}
	if got := field.String(); got != "(msg String): Error" {
		t.Fatalf("got %q", got)
	}
}

func TestShapeNode_IsMethodOnlyContract_emptyFields(t *testing.T) {
	if (ShapeNode{Fields: map[string]ShapeFieldNode{}}).IsMethodOnlyContract() {
		t.Fatal("empty shape is not method-only")
	}
}

func TestShapeFieldNode_ValueExpression_fromNode(t *testing.T) {
	lit := StringLiteralNode{Value: "wired"}
	field := ShapeFieldNode{Node: lit}
	expr, ok := field.ValueExpression()
	if !ok {
		t.Fatal("expected expression from Node")
	}
	if got, ok := expr.(StringLiteralNode); !ok || got.Value != "wired" {
		t.Fatalf("got %#v", expr)
	}
}

func TestShapeFieldNode_ValueExpression_nonExpressionNode(t *testing.T) {
	field := ShapeFieldNode{Node: CommentNode{Text: "note"}}
	if _, ok := field.ValueExpression(); ok {
		t.Fatal("comment node is not an expression")
	}
}

func TestShapeNode_String_field_assertion_branch(t *testing.T) {
	bt := TypeIdent("Str")
	s := ShapeNode{Fields: map[string]ShapeFieldNode{
		"k": {Assertion: &AssertionNode{BaseType: &bt}},
	}}
	out := s.String()
	if !strings.Contains(out, "k") || !strings.Contains(out, "Str") {
		t.Fatal(out)
	}
}
