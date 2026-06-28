package ast

import "testing"

func TestUseNode_KindAndString(t *testing.T) {
	named := UseNode{
		Ident:        &Ident{ID: "logger"},
		ContractType: TypeNode{Ident: "Logger"},
	}
	if named.Kind() != NodeKindUse {
		t.Fatalf("Kind() = %v, want NodeKindUse", named.Kind())
	}
	if named.String() != "Use(logger: Logger)" {
		t.Fatalf("String() = %q", named.String())
	}

	anon := UseNode{ContractType: TypeNode{Ident: "Clock"}}
	if anon.String() != "Use(Clock)" {
		t.Fatalf("anonymous String() = %q", anon.String())
	}
}

func TestWithNode_KindAndString(t *testing.T) {
	with := WithNode{
		Wiring: ShapeNode{Fields: map[string]ShapeFieldNode{}},
		Body:   []Node{},
	}
	if with.Kind() != NodeKindWith {
		t.Fatalf("Kind() = %v, want NodeKindWith", with.Kind())
	}
	if with.String() != "With({})" {
		t.Fatalf("String() = %q", with.String())
	}
}
