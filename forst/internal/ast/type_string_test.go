package ast

import (
	"strings"
	"testing"
)

func TestTypeIdent_String_all_known_tokens(t *testing.T) {
	cases := []struct {
		id   TypeIdent
		want string
	}{
		{TypeInt, "Int"},
		{TypeFloat, "Float"},
		{TypeString, "String"},
		{TypeBool, "Bool"},
		{TypeVoid, "Void"},
		{TypeError, "Error"},
		{TypeObject, "Object(?)"},
		{TypeArray, "Array(?)"},
		{TypeMap, "Map(?, ?)"},
		{TypeAssertion, "Assertion(?)"},
		{TypeImplicit, "(implicit)"},
		{TypeShape, "Shape(?)"},
		{TypePointer, "Pointer"},
		{TypeIdent("CustomType"), "CustomType"},
	}
	for _, tc := range cases {
		if got := tc.id.String(); got != tc.want {
			t.Errorf("TypeIdent(%q).String() = %q, want %q", tc.id, got, tc.want)
		}
	}
}

func TestTypeNode_String_branches(t *testing.T) {
	bt := NewBuiltinType(TypeInt)
	base := TypeIdent("User")
	assertion := &AssertionNode{BaseType: &base, Constraints: []ConstraintNode{{Name: "Min"}}}

	tests := []struct {
		name string
		typ  TypeNode
		sub  string
	}{
		{"builtin_int", TypeNode{Ident: TypeInt, TypeKind: TypeKindBuiltin}, "Int"},
		{"array_with_param", NewArrayType(bt), "Array("},
		{"array_empty_params", TypeNode{Ident: TypeArray}, "Array(?)"},
		{"map_full", NewMapType(NewBuiltinType(TypeString), NewBuiltinType(TypeInt)), "Map("},
		{"map_short", TypeNode{Ident: TypeMap}, "Map(?, ?)"},
		{"assertion_with_node", NewAssertionType(assertion), "Assertion("},
		{"assertion_empty", TypeNode{Ident: TypeAssertion}, "Assertion(?)"},
		{"implicit", TypeNode{Ident: TypeImplicit}, "(implicit)"},
		{"shape_with_param", TypeNode{Ident: TypeShape, TypeParams: []TypeNode{{Ident: TypeInt}}}, "Shape("},
		{"shape_no_params", TypeNode{Ident: TypeShape}, "Shape"},
		{"pointer_with_param", NewPointerType(NewBuiltinType(TypeString)), "Pointer("},
		{"pointer_empty", TypeNode{Ident: TypePointer}, "Pointer"},
		{"default_user_ident", TypeNode{Ident: "MyType", TypeKind: TypeKindUserDefined}, "MyType"},
		{"default_with_assertion", TypeNode{Ident: "X", Assertion: assertion}, "X("},
		{"default_with_type_params", TypeNode{Ident: "Box", TypeParams: []TypeNode{{Ident: TypeInt}}}, "Box<"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.typ.String()
			if !strings.Contains(s, tt.sub) {
				t.Fatalf("String() = %q, want substring %q", s, tt.sub)
			}
		})
	}
}

func TestTokenIdent_String_logical_and_bitwise(t *testing.T) {
	if TokenBitwiseAnd.String() != "&" || TokenBitwiseOr.String() != "|" {
		t.Fatal(TokenBitwiseAnd.String(), TokenBitwiseOr.String())
	}
	if TokenLogicalAnd.String() != "&&" || TokenLogicalOr.String() != "||" {
		t.Fatal(TokenLogicalAnd.String(), TokenLogicalOr.String())
	}
	// default branch
	if TokenPlus.String() != string(TokenPlus) {
		t.Fatalf("%q", TokenPlus.String())
	}
}
