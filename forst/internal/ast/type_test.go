package ast

import (
	"strings"
	"testing"
)

func TestTypeNode_builtin_hash_user_implicit_explicit_error(t *testing.T) {
	b := NewBuiltinType(TypeInt)
	if !b.IsGoBuiltin() || b.IsHashBased() || b.IsUserDefined() {
		t.Fatal()
	}
	h := NewHashBasedType("T_abc")
	if !h.IsHashBased() {
		t.Fatal()
	}
	ud := NewUserDefinedType("App")
	if !ud.IsUserDefined() {
		t.Fatal()
	}
	implicit := TypeNode{Ident: TypeImplicit}
	if !implicit.IsImplicit() || implicit.IsExplicit() {
		t.Fatal()
	}
	explicit := TypeNode{Ident: TypeInt}
	if !explicit.IsExplicit() || explicit.IsImplicit() {
		t.Fatal()
	}
	if !(TypeNode{Ident: TypeError}).IsError() {
		t.Fatal()
	}
}

func TestTypeIdent_String_custom_name_default_branch(t *testing.T) {
	if TypeIdent("ZZZ").String() != "ZZZ" {
		t.Fatal(TypeIdent("ZZZ").String())
	}
}

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
		{"default_with_type_params", TypeNode{Ident: "Box", TypeParams: []TypeNode{{Ident: TypeInt}}}, "Box("},
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

func TestTypeNode_String_type_object_case(t *testing.T) {
	s := TypeNode{Ident: TypeObject, TypeKind: TypeKindBuiltin}.String()
	if !strings.Contains(s, "Object") {
		t.Fatal(s)
	}
}

func TestTypeNode_String_each_simple_builtin_switch_case(t *testing.T) {
	cases := []struct {
		ident TypeIdent
		sub   string
	}{
		{TypeFloat, "Float"},
		{TypeString, "String"},
		{TypeBool, "Bool"},
		{TypeVoid, "Void"},
		{TypeError, "Error"},
	}
	for _, tc := range cases {
		s := TypeNode{Ident: tc.ident, TypeKind: TypeKindBuiltin}.String()
		if !strings.Contains(s, tc.sub) {
			t.Fatalf("Ident %v: String() = %q, want substring %q", tc.ident, s, tc.sub)
		}
	}
}

func TestTypeNode_String_default_branch_multiple_type_params(t *testing.T) {
	tn := TypeNode{
		Ident:    "Pair",
		TypeKind: TypeKindUserDefined,
		TypeParams: []TypeNode{
			{Ident: TypeInt, TypeKind: TypeKindBuiltin},
			{Ident: TypeString, TypeKind: TypeKindBuiltin},
		},
	}
	s := tn.String()
	if !strings.Contains(s, "Pair(") || !strings.Contains(s, "Int") || !strings.Contains(s, "String") {
		t.Fatal(s)
	}
}
