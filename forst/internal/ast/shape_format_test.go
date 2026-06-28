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

func TestShapeFieldNode_MethodSignatureString(t *testing.T) {
	field := ShapeFieldNode{
		IsMethod: true,
		MethodParams: []ParamNode{
			SimpleParamNode{Ident: Ident{ID: "n"}, Type: TypeNode{Ident: TypeInt}},
		},
	}
	if got := field.MethodSignatureString(); got != "(n Int)" {
		t.Fatalf("got %q", got)
	}
}

func TestShapeFieldNamesInOrder_usesFieldOrderWhenSet(t *testing.T) {
	fields := map[string]ShapeFieldNode{"b": {}, "a": {}}
	got := ShapeFieldNamesInOrder(fields, []string{"b", "a"})
	if len(got) != 2 || got[0] != "b" || got[1] != "a" {
		t.Fatalf("got %v", got)
	}
}

func TestShapeFieldNamesForCompositeEmit_skipsOrderEntriesMissingFromPayload(t *testing.T) {
	payload := map[string]ShapeFieldNode{"only": {Type: &TypeNode{Ident: TypeString}}}
	shape := &ShapeNode{FieldOrder: []string{"ghost", "only"}}
	got := ShapeFieldNamesForCompositeEmit(payload, shape)
	if len(got) != 1 || got[0] != "only" {
		t.Fatalf("got %v", got)
	}
}

func TestShapeFieldNamesForCompositeEmit_appendsExtraPayloadFieldsSorted(t *testing.T) {
	payload := map[string]ShapeFieldNode{
		"id":   {Type: &TypeNode{Ident: TypeString}},
		"extra": {Type: &TypeNode{Ident: TypeInt}},
	}
	shape := &ShapeNode{FieldOrder: []string{"id"}}
	got := ShapeFieldNamesForCompositeEmit(payload, shape)
	if len(got) != 2 || got[0] != "id" || got[1] != "extra" {
		t.Fatalf("got %v", got)
	}
}

func TestShapeFieldNamesForCompositeEmit_nilShapeUsesSortedOrder(t *testing.T) {
	payload := map[string]ShapeFieldNode{"z": {}, "a": {}}
	got := ShapeFieldNamesForCompositeEmit(payload, nil)
	if len(got) != 2 || got[0] != "a" || got[1] != "z" {
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
