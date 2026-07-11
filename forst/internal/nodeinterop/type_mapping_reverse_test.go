package nodeinterop

import (
	"testing"

	"forst/internal/ast"
)

func TestMapIndexTypeToForst_primitives(t *testing.T) {
	cases := []struct {
		kind ast.TypeIdent
		in   IndexType
	}{
		{ast.TypeString, IndexType{Kind: "string"}},
		{ast.TypeFloat, IndexType{Kind: "number"}},
		{ast.TypeBool, IndexType{Kind: "boolean"}},
		{ast.TypeVoid, IndexType{Kind: "void"}},
		{ast.TypeBytes, IndexType{Kind: "bytes"}},
		{ast.TypeObject, IndexType{Kind: "object"}},
	}

	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			got, _, err := MapIndexTypeToForst(tc.in)
			if err != nil {
				t.Fatalf("MapIndexTypeToForst: %v", err)
			}
			if got.Ident != tc.kind {
				t.Fatalf("Ident = %q, want %q", got.Ident, tc.kind)
			}
		})
	}
}

func TestMapIndexTypeToForst_objectWithFields(t *testing.T) {
	in := IndexType{
		Kind: "object",
		Fields: map[string]IndexType{
			"id":   {Kind: "string"},
			"amount": {Kind: "number"},
		},
	}

	got, _, err := MapIndexTypeToForst(in)
	if err != nil {
		t.Fatalf("MapIndexTypeToForst: %v", err)
	}
	if got.Ident != ast.TypeAssertion {
		t.Fatalf("Ident = %q, want TYPE_ASSERTION", got.Ident)
	}
	if got.Assertion == nil || len(got.Assertion.Constraints) == 0 {
		t.Fatal("expected Shape assertion")
	}
	shape := got.Assertion.Constraints[0].Args[0].Shape
	if shape == nil {
		t.Fatal("expected shape argument")
	}
	if len(shape.Fields) != 2 {
		t.Fatalf("len(fields) = %d, want 2", len(shape.Fields))
	}
	idField, ok := shape.Fields["id"]
	if !ok || idField.Type == nil || idField.Type.Ident != ast.TypeString {
		t.Fatalf("id field = %+v, want string type", idField)
	}
}

func TestMapIndexTypeToForst_unsupportedKind(t *testing.T) {
	_, _, err := MapIndexTypeToForst(IndexType{Kind: "effect"})
	if err == nil {
		t.Fatal("expected error for unsupported kind")
	}
}

func TestMapIndexTypeToForst_unknown(t *testing.T) {
	got, widened, err := MapIndexTypeToForst(IndexType{Kind: "unknown"})
	if err != nil {
		t.Fatalf("MapIndexTypeToForst: %v", err)
	}
	if got.Ident != ast.TypeObject {
		t.Fatalf("Ident = %q, want %q", got.Ident, ast.TypeObject)
	}
	if !widened {
		t.Fatal("expected widened flag for unknown")
	}
}

func TestMapIndexTypeToForst_union(t *testing.T) {
	in := IndexType{
		Kind: "union",
		Members: []IndexType{
			{Kind: "string"},
			{Kind: "number"},
		},
	}

	got, _, err := MapIndexTypeToForst(in)
	if err != nil {
		t.Fatalf("MapIndexTypeToForst: %v", err)
	}
	if got.Ident != ast.TypeUnion {
		t.Fatalf("Ident = %q, want %q", got.Ident, ast.TypeUnion)
	}
	if len(got.TypeParams) != 2 {
		t.Fatalf("len(TypeParams) = %d, want 2", len(got.TypeParams))
	}
	if got.TypeParams[0].Ident != ast.TypeString {
		t.Fatalf("member[0] = %q, want string", got.TypeParams[0].Ident)
	}
	if got.TypeParams[1].Ident != ast.TypeFloat {
		t.Fatalf("member[1] = %q, want float", got.TypeParams[1].Ident)
	}
}

func TestMapIndexTypeToForst_unionSingleMemberCollapses(t *testing.T) {
	in := IndexType{
		Kind:    "union",
		Members: []IndexType{{Kind: "boolean"}},
	}

	got, _, err := MapIndexTypeToForst(in)
	if err != nil {
		t.Fatalf("MapIndexTypeToForst: %v", err)
	}
	if got.Ident != ast.TypeBool {
		t.Fatalf("Ident = %q, want bool (single-member union collapse)", got.Ident)
	}
}

func TestMapIndexTypeToForst_unionEmptyMembers(t *testing.T) {
	got, widened, err := MapIndexTypeToForst(IndexType{Kind: "union", Members: nil})
	if err != nil {
		t.Fatalf("MapIndexTypeToForst: %v", err)
	}
	if got.Ident != ast.TypeObject {
		t.Fatalf("Ident = %q, want %q", got.Ident, ast.TypeObject)
	}
	if !widened {
		t.Fatal("expected widened flag for empty union")
	}
}

func TestMapIndexTypeToForst_unionUnsupportedMemberWidensToObject(t *testing.T) {
	in := IndexType{
		Kind: "union",
		Members: []IndexType{
			{Kind: "string"},
			{Kind: "effect"},
		},
	}

	got, _, err := MapIndexTypeToForst(in)
	if err != nil {
		t.Fatalf("MapIndexTypeToForst: %v", err)
	}
	if got.Ident != ast.TypeObject {
		t.Fatalf("Ident = %q, want %q", got.Ident, ast.TypeObject)
	}
}

func TestMapIndexTypeToForst_unionWithUnknownMember(t *testing.T) {
	in := IndexType{
		Kind: "union",
		Members: []IndexType{
			{Kind: "string"},
			{Kind: "unknown"},
		},
	}

	got, _, err := MapIndexTypeToForst(in)
	if err != nil {
		t.Fatalf("MapIndexTypeToForst: %v", err)
	}
	if got.Ident != ast.TypeUnion {
		t.Fatalf("Ident = %q, want %q", got.Ident, ast.TypeUnion)
	}
	if len(got.TypeParams) != 2 {
		t.Fatalf("len(TypeParams) = %d, want 2", len(got.TypeParams))
	}
	if got.TypeParams[0].Ident != ast.TypeString {
		t.Fatalf("member[0] = %q, want string", got.TypeParams[0].Ident)
	}
	if got.TypeParams[1].Ident != ast.TypeObject {
		t.Fatalf("member[1] = %q, want object (unknown escape hatch)", got.TypeParams[1].Ident)
	}
}
