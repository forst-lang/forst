package ast

import (
	"strings"
	"testing"
)

func TestReferenceNode_String_and_GetIdent(t *testing.T) {
	r := ReferenceNode{Value: VariableNode{Ident: Ident{ID: "z"}}}
	if r.Kind() != NodeKindReference {
		t.Fatal(r.Kind())
	}
	if r.GetIdent() != r.Value.String() || !strings.Contains(r.String(), "z") {
		t.Fatal(r.String(), r.GetIdent())
	}
}

func TestReferenceNode_isValue_marker(_ *testing.T) {
	ReferenceNode{}.isValue()
}
