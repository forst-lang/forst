package ast

import (
	"strings"
	"testing"
)

func TestArrayLiteralNode_String_multi_element(t *testing.T) {
	arr := ArrayLiteralNode{
		Value: []LiteralNode{IntLiteralNode{Value: 1}, IntLiteralNode{Value: 2}},
		Type:  NewBuiltinType(TypeInt),
	}
	if !strings.Contains(arr.String(), "1") || !strings.Contains(arr.String(), "2") {
		t.Fatal(arr.String())
	}
}

func TestMapLiteralNode_String(t *testing.T) {
	mapTyp := NewMapType(NewBuiltinType(TypeString), NewBuiltinType(TypeInt))
	m := MapLiteralNode{
		Type: mapTyp,
		Entries: []MapEntryNode{
			{Key: StringLiteralNode{Value: "k"}, Value: IntLiteralNode{Value: 7}},
		},
	}
	ms := m.String()
	if !strings.Contains(ms, "k") || !strings.Contains(ms, "7") || !strings.Contains(ms, "map[") {
		t.Fatal(ms)
	}
}

func TestLiteralNodes_marker_methods_and_array_map_Kind(t *testing.T) {
	IntLiteralNode{}.isLiteral()
	FloatLiteralNode{}.isLiteral()
	StringLiteralNode{}.isLiteral()
	BoolLiteralNode{}.isLiteral()
	ArrayLiteralNode{}.isLiteral()
	MapLiteralNode{}.isLiteral()
	NilLiteralNode{}.isLiteral()

	IntLiteralNode{}.isValue()
	FloatLiteralNode{}.isValue()
	StringLiteralNode{}.isValue()
	BoolLiteralNode{}.isValue()
	ArrayLiteralNode{}.isValue()
	MapLiteralNode{}.isValue()
	NilLiteralNode{}.isValue()

	IntLiteralNode{}.isExpression()
	FloatLiteralNode{}.isExpression()
	StringLiteralNode{}.isExpression()
	BoolLiteralNode{}.isExpression()
	ArrayLiteralNode{}.isExpression()
	MapLiteralNode{}.isExpression()
	NilLiteralNode{}.isExpression()

	if (ArrayLiteralNode{}).Kind() != NodeKindArrayLiteral {
		t.Fatal()
	}
	if (MapLiteralNode{}).Kind() != NodeKindMapLiteral {
		t.Fatal()
	}
}
