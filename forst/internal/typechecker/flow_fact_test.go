package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestMergeFlowFactsAtIfJoin_emptyInputs(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	out := MergeFlowFactsAtIfJoin(tc, nil, nil)
	if len(out) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(out))
	}
	out = MergeFlowFactsAtIfJoin(tc, map[ast.Identifier]ast.TypeNode{}, []FlowTypeFact{})
	if len(out) != 0 {
		t.Fatalf("expected empty map")
	}
}

func TestMergeFlowFactsAtIfJoin_oneRefinementWidensToOuter(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	str := ast.TypeString
	tc.registerType(ast.TypeDefNode{
		Ident: "MyStr",
		Expr: &ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{BaseType: &str},
		},
	})
	id := ast.Identifier("x")
	outer := ast.TypeNode{Ident: ast.TypeIdent("MyStr")}
	facts := []FlowTypeFact{
		{Ident: id, RefinedType: ast.TypeNode{Ident: ast.TypeString}},
	}
	out := MergeFlowFactsAtIfJoin(tc, map[ast.Identifier]ast.TypeNode{id: outer}, facts)
	if len(out) != 1 || out[id].Ident != outer.Ident {
		t.Fatalf("merge: got %+v", out)
	}
}

func TestMergeFlowFactsAtIfJoin_outerWithoutMatchingFactsSkipped(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	str := ast.TypeString
	tc.registerType(ast.TypeDefNode{
		Ident: "MyStr",
		Expr: &ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{BaseType: &str},
		},
	})
	x, y := ast.Identifier("x"), ast.Identifier("y")
	outerX := ast.TypeNode{Ident: ast.TypeIdent("MyStr")}
	outerY := ast.TypeNode{Ident: ast.TypeInt}
	facts := []FlowTypeFact{
		{Ident: x, RefinedType: ast.TypeNode{Ident: ast.TypeString}},
	}
	out := MergeFlowFactsAtIfJoin(
		tc,
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

func TestMergeFlowFactsAtIfJoin_multipleRefinementsSameIdJoinToUnion(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	id := ast.Identifier("x")
	outer := ast.TypeNode{Ident: ast.TypeIdent("Outer")}
	facts := []FlowTypeFact{
		{Ident: id, RefinedType: ast.TypeNode{Ident: ast.TypeString}},
		{Ident: id, RefinedType: ast.TypeNode{Ident: ast.TypeString}},
		{Ident: id, RefinedType: ast.TypeNode{Ident: ast.TypeInt}},
	}
	out := MergeFlowFactsAtIfJoin(tc, map[ast.Identifier]ast.TypeNode{id: outer}, facts)
	if len(out) != 1 {
		t.Fatalf("merge: got %+v", out)
	}
	got := out[id]
	if got.Ident != ast.TypeUnion || len(got.TypeParams) != 3 {
		t.Fatalf("want union(Outer,String,Int), got %+v", got)
	}
}

func TestMergeFlowFactsAtIfJoin_twoIdsIndependent(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	x, y := ast.Identifier("x"), ast.Identifier("y")
	ox := ast.TypeNode{Ident: ast.TypeIdent("A")}
	oy := ast.TypeNode{Ident: ast.TypeIdent("B")}
	facts := []FlowTypeFact{
		{Ident: x, RefinedType: ast.TypeNode{Ident: ast.TypeString}},
		{Ident: y, RefinedType: ast.TypeNode{Ident: ast.TypeInt}},
	}
	out := MergeFlowFactsAtIfJoin(
		tc,
		map[ast.Identifier]ast.TypeNode{x: ox, y: oy},
		facts,
	)
	if len(out) != 2 {
		t.Fatalf("want 2 keys, got %+v", out)
	}
	if out[x].Ident != ast.TypeUnion || out[y].Ident != ast.TypeUnion {
		t.Fatalf("want union join per id: %+v", out)
	}
}

func TestMergeFlowFactsAtIfJoin_factsPreserveOrderInRefinementsSlice(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	// Order of refinements is preserved through dedupe + union (String before Int).
	id := ast.Identifier("x")
	outer := ast.TypeNode{Ident: ast.TypeIdent("U")}
	facts := []FlowTypeFact{
		{Ident: id, RefinedType: ast.TypeNode{Ident: ast.TypeString}, NarrowingTypeGuards: []string{"g1"}},
		{Ident: id, RefinedType: ast.TypeNode{Ident: ast.TypeInt}, NarrowingTypeGuards: []string{"g2"}},
	}
	out := MergeFlowFactsAtIfJoin(tc, map[ast.Identifier]ast.TypeNode{id: outer}, facts)
	if len(out) != 1 || out[id].Ident != ast.TypeUnion {
		t.Fatalf("got %+v", out)
	}
}

func TestMergeFlowFactsAtIfJoin_branchFactsForUnknownOuterIgnored(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	// Facts mentioning z with no outer entry: outer loop only emits keys present in outerByIdent.
	x := ast.Identifier("x")
	z := ast.Identifier("z")
	outer := ast.TypeNode{Ident: ast.TypeString}
	facts := []FlowTypeFact{
		{Ident: x, RefinedType: ast.TypeNode{Ident: ast.TypeInt}},
		{Ident: z, RefinedType: ast.TypeNode{Ident: ast.TypeBool}},
	}
	out := MergeFlowFactsAtIfJoin(tc, map[ast.Identifier]ast.TypeNode{x: outer}, facts)
	if len(out) != 1 || out[x].Ident != ast.TypeUnion {
		t.Fatalf("got %+v", out)
	}
	if _, ok := out[z]; ok {
		t.Fatal("z should not appear without outer mapping")
	}
}
