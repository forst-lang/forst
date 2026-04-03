package ast

import (
	"strings"
	"testing"
)

func TestDestructuredParamNode_String_Kind_GetType(t *testing.T) {
	d := DestructuredParamNode{
		Fields: []string{"a", "b"},
		Type:   NewBuiltinType(TypeInt),
	}
	if d.Kind() != NodeKindDestructuredParam || !strings.Contains(d.String(), "a") {
		t.Fatal(d.String())
	}
	if d.GetType().Ident != TypeInt {
		t.Fatal(d.GetType())
	}
}
