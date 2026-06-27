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
