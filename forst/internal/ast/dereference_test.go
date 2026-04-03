package ast

import (
	"strings"
	"testing"
)

func TestDereferenceNode_String(t *testing.T) {
	d := DereferenceNode{Value: VariableNode{Ident: Ident{ID: "p"}}}
	if d.Kind() != NodeKindDereference || !strings.Contains(d.String(), "p") {
		t.Fatal(d.String(), d.Kind())
	}
}
