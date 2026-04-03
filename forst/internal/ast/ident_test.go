package ast

import "testing"

func TestIdent_kind_and_string(t *testing.T) {
	id := Ident{ID: "foo"}
	if id.Kind() != NodeKindIdentifier {
		t.Fatal(id.Kind())
	}
	if id.String() != "foo" {
		t.Fatal(id.String())
	}
}
