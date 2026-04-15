package ast

import (
	"testing"
)

func TestNewResultType_andNewTupleType(t *testing.T) {
	t.Parallel()
	r := NewResultType(TypeNode{Ident: TypeInt}, TypeNode{Ident: TypeError})
	if r.Ident != TypeResult || len(r.TypeParams) != 2 {
		t.Fatalf("got %+v", r)
	}
	tup := NewTupleType(TypeNode{Ident: TypeInt}, TypeNode{Ident: TypeString})
	if tup.Ident != TypeTuple || len(tup.TypeParams) != 2 {
		t.Fatalf("got %+v", tup)
	}
}

func TestNewUnionType_flattensNestedUnions(t *testing.T) {
	t.Parallel()
	inner := TypeNode{Ident: TypeUnion, TypeParams: []TypeNode{{Ident: TypeInt}, {Ident: TypeString}}}
	u := NewUnionType(inner, TypeNode{Ident: TypeBool})
	if u.Ident != TypeUnion {
		t.Fatalf("got %+v", u)
	}
	if len(u.TypeParams) != 3 {
		t.Fatalf("expected flattened 3 members, got %d: %+v", len(u.TypeParams), u.TypeParams)
	}
}

func TestNewUnionType_singleMemberReturnsUnderlying(t *testing.T) {
	t.Parallel()
	only := TypeNode{Ident: TypeInt}
	u := NewUnionType(only)
	if u.Ident != TypeInt {
		t.Fatalf("expected passthrough, got %+v", u)
	}
}

func TestNewIntersectionType_flattensNested(t *testing.T) {
	t.Parallel()
	inner := TypeNode{Ident: TypeIntersection, TypeParams: []TypeNode{{Ident: TypeInt}, {Ident: TypeString}}}
	u := NewIntersectionType(inner, TypeNode{Ident: TypeBool})
	if u.Ident != TypeIntersection || len(u.TypeParams) != 3 {
		t.Fatalf("got %+v", u)
	}
}

func TestNewUnionType_dedupesShallowEqualMembers(t *testing.T) {
	t.Parallel()
	a := TypeNode{Ident: TypeInt}
	u := NewUnionType(a, a, a)
	if u.Ident != TypeInt {
		t.Fatalf("expected passthrough to single Int, got %+v", u)
	}
}
