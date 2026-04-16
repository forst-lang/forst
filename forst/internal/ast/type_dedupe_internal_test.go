package ast

import "testing"

func TestDedupeTypeNodesShallow_emptyInput(t *testing.T) {
	t.Parallel()
	if dedupeTypeNodesShallow(nil) != nil {
		t.Fatal("nil slice should dedupe to nil")
	}
	if dedupeTypeNodesShallow([]TypeNode{}) != nil {
		t.Fatal("empty slice should dedupe to nil")
	}
}

func TestTypeNodesShallowEqualAST_falseCases(t *testing.T) {
	t.Parallel()
	if typeNodesShallowEqualAST(NewBuiltinType(TypeInt), NewBuiltinType(TypeString)) {
		t.Fatal("different idents")
	}
	if typeNodesShallowEqualAST(
		TypeNode{Ident: TypeUnion, TypeParams: []TypeNode{{Ident: TypeInt}}},
		TypeNode{Ident: TypeUnion, TypeParams: []TypeNode{{Ident: TypeInt}, {Ident: TypeString}}},
	) {
		t.Fatal("same ident, different type param count")
	}
	if typeNodesShallowEqualAST(
		NewArrayType(NewBuiltinType(TypeInt)),
		NewArrayType(NewBuiltinType(TypeString)),
	) {
		t.Fatal("recursive param mismatch")
	}
}
