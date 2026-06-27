package hasher

import (
	"testing"

	"forst/internal/ast"
)

func TestHashSortedStrings_orderIndependent(t *testing.T) {
	a := HashSortedStrings("Clock", "Logger")
	b := HashSortedStrings("Logger", "Clock")
	if a != b {
		t.Fatalf("slot-set hash should be order-independent: %v vs %v", a, b)
	}
}

func TestHashSortedStrings_distinctSets(t *testing.T) {
	one := HashSortedStrings("Logger")
	two := HashSortedStrings("Logger", "Clock")
	if one == two {
		t.Fatal("different slot sets should produce different hashes")
	}
}

func TestNodeHash_ToProvidersIdent_prefix(t *testing.T) {
	h := HashSortedStrings("Logger", "Clock")
	id := h.ToProvidersIdent()
	if !stringsHasPrefix(id, "Providers_") {
		t.Fatalf("ToProvidersIdent() = %q, want Providers_ prefix", id)
	}
	if len(id) <= len("Providers_") {
		t.Fatalf("ToProvidersIdent() too short: %q", id)
	}
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func TestHashNode_UseNode_namedAndAnonymousDistinct(t *testing.T) {
	h := New()
	named := ast.UseNode{
		Ident:        &ast.Ident{ID: "logger"},
		ContractType: ast.TypeNode{Ident: "Logger"},
	}
	anon := ast.UseNode{ContractType: ast.TypeNode{Ident: "Logger"}}
	hNamed, err := h.HashNode(named)
	if err != nil {
		t.Fatalf("HashNode named use: %v", err)
	}
	hAnon, err := h.HashNode(anon)
	if err != nil {
		t.Fatalf("HashNode anonymous use: %v", err)
	}
	if hNamed == hAnon {
		t.Fatal("named and anonymous use nodes should hash differently for scope restoration")
	}
}

func TestHashNode_WithNode_includesBody(t *testing.T) {
	h := New()
	w1 := ast.WithNode{
		Wiring: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{"Logger": {}}},
		Body:   []ast.Node{ast.UseNode{ContractType: ast.TypeNode{Ident: "Logger"}}},
	}
	w2 := ast.WithNode{
		Wiring: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{"Logger": {}}},
		Body:   []ast.Node{},
	}
	h1, err := h.HashNode(w1)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := h.HashNode(w2)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Fatal("with blocks with different bodies should hash differently")
	}
}
