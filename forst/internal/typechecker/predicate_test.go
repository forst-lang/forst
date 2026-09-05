package typechecker

import (
	"strings"
	"testing"
)

func TestPredicate_flattenSortDedupe(t *testing.T) {
	pi := NewPredicateInterner()
	a := pi.InternAtom(Operand{Name: "B"})
	b := pi.InternAtom(Operand{Name: "A"})
	c := pi.InternAtom(Operand{Name: "A"})
	all := pi.InternAll([]*Predicate{
		pi.InternAll([]*Predicate{a, b}),
		c,
	})
	// Flattened All, sorted by key, A deduped → All(A,B)
	if all.Conn != PredAll || len(all.Children) != 2 {
		t.Fatalf("shape=%s conn=%d kids=%d", all.Shape(), all.Conn, len(all.Children))
	}
	if all.Children[0].Operand.Name != "A" || all.Children[1].Operand.Name != "B" {
		t.Fatalf("sort/dedupe failed: %s", all.Shape())
	}
	all2 := pi.InternAll([]*Predicate{b, a})
	if all != all2 {
		t.Fatal("same All must intern to identical pointer")
	}
}

func TestPredicate_noDNFDistribution(t *testing.T) {
	pi := NewPredicateInterner()
	// All(Any(A,B), C) must NOT become Any(All(A,C), All(B,C))
	ir := All{
		Children: []Assertion{
			Any{Children: []Assertion{Atom{Name: "A"}, Atom{Name: "B"}}},
			Atom{Name: "C"},
		},
	}
	p := pi.FromAssertion(ir)
	shape := p.Shape()
	if shape != "All(Any(Atom(A),Atom(B)),Atom(C))" && shape != "All(Atom(C),Any(Atom(A),Atom(B)))" {
		// sort may reorder All children by key: any:… vs atom:C
		if !strings.Contains(shape, "Any(") || !strings.Contains(shape, "Atom(C)") || strings.HasPrefix(shape, "Any(") {
			t.Fatalf("no DNF: got %s", shape)
		}
	}
	if p.Conn == PredAny {
		t.Fatalf("distributed to top-level Any: %s", shape)
	}
}

func TestPredicate_stableKeys(t *testing.T) {
	pi := NewPredicateInterner()
	p1 := pi.FromAssertion(Any{Children: []Assertion{Atom{Name: "B"}, Atom{Name: "A"}}})
	p2 := pi.FromAssertion(Any{Children: []Assertion{Atom{Name: "A"}, Atom{Name: "B"}}})
	if p1 != p2 {
		t.Fatalf("order-independent Any keys: %q vs %q", p1.Key(), p2.Key())
	}
	if p1.Key() == "" {
		t.Fatal("empty key")
	}
}
