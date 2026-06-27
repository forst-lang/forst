package ast

import "testing"

func TestFormatShapeMemberName_methodOmitsColon(t *testing.T) {
	field := ShapeFieldNode{
		IsMethod: true,
		MethodParams: []ParamNode{
			SimpleParamNode{
				Ident: Ident{ID: "msg"},
				Type:  TypeNode{Ident: TypeString},
			},
		},
	}
	got := FormatShapeMemberName("info", field)
	want := "info(msg String)"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatShapeMemberName_dataFieldUsesColon(t *testing.T) {
	field := ShapeFieldNode{
		Type: &TypeNode{Ident: TypeString},
	}
	got := FormatShapeMemberName("id", field)
	if got != "id: String" {
		t.Fatalf("got %q", got)
	}
}

func TestShapeFieldNamesForCompositeEmit_preservesShapeFieldOrder(t *testing.T) {
	payload := map[string]ShapeFieldNode{
		"id":        {Type: &TypeNode{Ident: TypeString}},
		"expiresAt": {Type: &TypeNode{Ident: TypeInt}},
	}
	shape := &ShapeNode{
		FieldOrder: []string{"id", "expiresAt"},
		Fields:     payload,
	}
	got := ShapeFieldNamesForCompositeEmit(payload, shape)
	if len(got) != 2 || got[0] != "id" || got[1] != "expiresAt" {
		t.Fatalf("got %v", got)
	}
}

func TestShapeFieldNamesInOrder_sortsWhenUnspecified(t *testing.T) {
	fields := map[string]ShapeFieldNode{
		"z": {},
		"a": {},
	}
	got := ShapeFieldNamesInOrder(fields, nil)
	if len(got) != 2 || got[0] != "a" || got[1] != "z" {
		t.Fatalf("got %v", got)
	}
}
