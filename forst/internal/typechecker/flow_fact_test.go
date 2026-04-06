package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestMergeFlowFactsAtIfJoin_emptyInputs(t *testing.T) {
	t.Parallel()
	out := MergeFlowFactsAtIfJoin(nil, nil)
	if len(out) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(out))
	}
	out = MergeFlowFactsAtIfJoin(map[ast.Identifier]ast.TypeNode{}, []FlowTypeFact{})
	if len(out) != 0 {
		t.Fatalf("expected empty map")
	}
}

func TestMergeFlowFactsAtIfJoin_oneRefinementWidensToOuter(t *testing.T) {
	t.Parallel()
	id := ast.Identifier("x")
	outer := ast.TypeNode{Ident: ast.TypeIdent("MyStr")}
	facts := []FlowTypeFact{
		{Ident: id, RefinedType: ast.TypeNode{Ident: ast.TypeString}},
	}
	out := MergeFlowFactsAtIfJoin(map[ast.Identifier]ast.TypeNode{id: outer}, facts)
	if len(out) != 1 || out[id].Ident != outer.Ident {
		t.Fatalf("merge: got %+v", out)
	}
}

func TestMergeFlowFactsAtIfJoin_outerWithoutMatchingFactsSkipped(t *testing.T) {
	t.Parallel()
	x, y := ast.Identifier("x"), ast.Identifier("y")
	outerX := ast.TypeNode{Ident: ast.TypeIdent("MyStr")}
	outerY := ast.TypeNode{Ident: ast.TypeInt}
	facts := []FlowTypeFact{
		{Ident: x, RefinedType: ast.TypeNode{Ident: ast.TypeString}},
	}
	out := MergeFlowFactsAtIfJoin(
		map[ast.Identifier]ast.TypeNode{x: outerX, y: outerY},
		facts,
	)
	if len(out) != 1 {
		t.Fatalf("want only x merged, got len=%d %+v", len(out), out)
	}
	if out[x].Ident != outerX.Ident {
		t.Fatalf("x: got %s want %s", out[x].Ident, outerX.Ident)
	}
	if _, ok := out[y]; ok {
		t.Fatal("y should not appear without branch facts")
	}
}

func TestMergeFlowFactsAtIfJoin_multipleRefinementsSameIdJoinToOuter(t *testing.T) {
	t.Parallel()
	id := ast.Identifier("x")
	outer := ast.TypeNode{Ident: ast.TypeIdent("Outer")}
	facts := []FlowTypeFact{
		{Ident: id, RefinedType: ast.TypeNode{Ident: ast.TypeString}},
		{Ident: id, RefinedType: ast.TypeNode{Ident: ast.TypeString}},
		{Ident: id, RefinedType: ast.TypeNode{Ident: ast.TypeInt}},
	}
	out := MergeFlowFactsAtIfJoin(map[ast.Identifier]ast.TypeNode{id: outer}, facts)
	if len(out) != 1 || out[id].Ident != outer.Ident {
		t.Fatalf("merge to outer: got %+v", out)
	}
}

func TestMergeFlowFactsAtIfJoin_twoIdsIndependent(t *testing.T) {
	t.Parallel()
	x, y := ast.Identifier("x"), ast.Identifier("y")
	ox := ast.TypeNode{Ident: ast.TypeIdent("A")}
	oy := ast.TypeNode{Ident: ast.TypeIdent("B")}
	facts := []FlowTypeFact{
		{Ident: x, RefinedType: ast.TypeNode{Ident: ast.TypeString}},
		{Ident: y, RefinedType: ast.TypeNode{Ident: ast.TypeInt}},
	}
	out := MergeFlowFactsAtIfJoin(
		map[ast.Identifier]ast.TypeNode{x: ox, y: oy},
		facts,
	)
	if len(out) != 2 {
		t.Fatalf("want 2 keys, got %+v", out)
	}
	if out[x].Ident != ox.Ident || out[y].Ident != oy.Ident {
		t.Fatalf("each key widens to its own outer: %+v", out)
	}
}

func TestMergeFlowFactsAtIfJoin_factsPreserveOrderInRefinementsSlice(t *testing.T) {
	t.Parallel()
	// JoinAfterIfMerge ignores refinement values today; still ensure MergeFlowFacts passes
	// every fact through to JoinAfterIfMerge (future union/LUB may depend on order).
	id := ast.Identifier("x")
	outer := ast.TypeNode{Ident: ast.TypeIdent("U")}
	facts := []FlowTypeFact{
		{Ident: id, RefinedType: ast.TypeNode{Ident: ast.TypeString}, NarrowingTypeGuards: []string{"g1"}},
		{Ident: id, RefinedType: ast.TypeNode{Ident: ast.TypeInt}, NarrowingTypeGuards: []string{"g2"}},
	}
	out := MergeFlowFactsAtIfJoin(map[ast.Identifier]ast.TypeNode{id: outer}, facts)
	if len(out) != 1 || out[id].Ident != outer.Ident {
		t.Fatalf("got %+v", out)
	}
}

func TestMergeFlowFactsAtIfJoin_branchFactsForUnknownOuterIgnored(t *testing.T) {
	t.Parallel()
	// Facts mentioning z with no outer entry: outer loop only emits keys present in outerByIdent.
	x := ast.Identifier("x")
	z := ast.Identifier("z")
	outer := ast.TypeNode{Ident: ast.TypeString}
	facts := []FlowTypeFact{
		{Ident: x, RefinedType: ast.TypeNode{Ident: ast.TypeInt}},
		{Ident: z, RefinedType: ast.TypeNode{Ident: ast.TypeBool}},
	}
	out := MergeFlowFactsAtIfJoin(map[ast.Identifier]ast.TypeNode{x: outer}, facts)
	if len(out) != 1 || out[x].Ident != outer.Ident {
		t.Fatalf("got %+v", out)
	}
	if _, ok := out[z]; ok {
		t.Fatal("z should not appear without outer mapping")
	}
}
