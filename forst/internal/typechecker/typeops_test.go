package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestWidenToEnclosing_isIdentity(t *testing.T) {
	t.Parallel()
	outer := ast.TypeNode{Ident: ast.TypeIdent("T")}
	if got := WidenToEnclosing(outer); got.Ident != outer.Ident {
		t.Fatalf("WidenToEnclosing: %+v", got)
	}
}

func TestJoinAfterIfMerge_table(t *testing.T) {
	t.Parallel()
	tc := New(discardLogger(), false)
	outer := ast.TypeNode{Ident: ast.TypeIdent("Outer")}
	cases := []struct {
		name string
		ref  []ast.TypeNode
		want func(ast.TypeNode) bool
	}{
		{"nil_refinements", nil, func(g ast.TypeNode) bool { return g.Ident == outer.Ident }},
		{"empty_refinements", []ast.TypeNode{}, func(g ast.TypeNode) bool { return g.Ident == outer.Ident }},
		{"single_refinement", []ast.TypeNode{{Ident: ast.TypeString}}, func(g ast.TypeNode) bool {
			return g.Ident == ast.TypeUnion && len(g.TypeParams) == 2
		}},
		{"multiple_refinements", []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeInt}}, func(g ast.TypeNode) bool {
			return g.Ident == ast.TypeUnion && len(g.TypeParams) == 3
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := JoinAfterIfMerge(tc, outer, c.ref)
			if !c.want(got) {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

func TestMeetTypes_equalIdents(t *testing.T) {
	t.Parallel()
	a := ast.TypeNode{Ident: ast.TypeString}
	b := ast.TypeNode{Ident: ast.TypeString}
	got, ok := MeetTypes(a, b)
	if !ok || got.Ident != ast.TypeString {
		t.Fatalf("MeetTypes: %+v ok=%v", got, ok)
	}
}

func TestMeetTypes_distinct(t *testing.T) {
	t.Parallel()
	a := ast.TypeNode{Ident: ast.TypeString}
	b := ast.TypeNode{Ident: ast.TypeInt}
	_, ok := MeetTypes(a, b)
	if ok {
		t.Fatal("expected false for distinct types (stub)")
	}
}

func TestMeetTypes_sameIdentMismatchedTypeParams(t *testing.T) {
	t.Parallel()
	a := ast.TypeNode{Ident: ast.TypeIdent("Box"), TypeParams: []ast.TypeNode{{Ident: ast.TypeString}}}
	b := ast.TypeNode{Ident: ast.TypeIdent("Box"), TypeParams: []ast.TypeNode{{Ident: ast.TypeInt}}}
	_, ok := MeetTypes(a, b)
	if ok {
		t.Fatal("expected false when TypeParams differ")
	}
}

func TestMeetTypes_nestedTypeParamsMatch(t *testing.T) {
	t.Parallel()
	inner := func(id ast.TypeIdent) ast.TypeNode {
		return ast.TypeNode{Ident: ast.TypeIdent("List"), TypeParams: []ast.TypeNode{{Ident: id}}}
	}
	a := inner(ast.TypeString)
	b := inner(ast.TypeString)
	got, ok := MeetTypes(a, b)
	if !ok || got.Ident != a.Ident {
		t.Fatalf("MeetTypes nested: %+v ok=%v", got, ok)
	}
}

func TestMeetTypes_table(t *testing.T) {
	t.Parallel()
	box := ast.TypeIdent("Box")
	cases := []struct {
		name string
		a, b ast.TypeNode
		ok   bool
	}{
		{
			name: "both_builtin_string",
			a:    ast.TypeNode{Ident: ast.TypeString},
			b:    ast.TypeNode{Ident: ast.TypeString},
			ok:   true,
		},
		{
			name: "deep_equal_params",
			a: ast.TypeNode{
				Ident: box,
				TypeParams: []ast.TypeNode{
					{Ident: ast.TypeIdent("Pair"), TypeParams: []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeInt}}},
				},
			},
			b: ast.TypeNode{
				Ident: box,
				TypeParams: []ast.TypeNode{
					{Ident: ast.TypeIdent("Pair"), TypeParams: []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeInt}}},
				},
			},
			ok: true,
		},
		{
			name: "param_count_mismatch",
			a:    ast.TypeNode{Ident: box, TypeParams: []ast.TypeNode{{Ident: ast.TypeString}}},
			b:    ast.TypeNode{Ident: box, TypeParams: []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeInt}}},
			ok:   false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, ok := MeetTypes(c.a, c.b)
			if ok != c.ok {
				t.Fatalf("MeetTypes ok=%v want %v", ok, c.ok)
			}
		})
	}
}

func TestJoinTypes_nary(t *testing.T) {
	t.Parallel()
	types := []ast.TypeNode{
		{Ident: ast.TypeString},
		{Ident: ast.TypeString},
	}
	got, ok := JoinTypes(types)
	if !ok || got.Ident != ast.TypeString {
		t.Fatalf("JoinTypes: %+v", got)
	}
}

func TestJoinTypes_heterogeneous(t *testing.T) {
	t.Parallel()
	types := []ast.TypeNode{
		{Ident: ast.TypeString},
		{Ident: ast.TypeInt},
	}
	got, ok := JoinTypes(types)
	if !ok || got.Ident != ast.TypeUnion || len(got.TypeParams) != 2 {
		t.Fatalf("JoinTypes heterogeneous: got %+v ok=%v", got, ok)
	}
}

func TestJoinTypes_empty(t *testing.T) {
	t.Parallel()
	got, ok := JoinTypes(nil)
	if ok || got.Ident != "" {
		t.Fatalf("JoinTypes empty: got %+v ok=%v", got, ok)
	}
	got, ok = JoinTypes([]ast.TypeNode{})
	if ok || got.Ident != "" {
		t.Fatalf("JoinTypes empty slice: got %+v ok=%v", got, ok)
	}
}

func TestJoinTypes_heterogeneousFirstStubAfterDedupe(t *testing.T) {
	t.Parallel()
	// [String, Int, String] → dedupe [String, Int] → union IR.
	got, ok := JoinTypes([]ast.TypeNode{
		{Ident: ast.TypeString},
		{Ident: ast.TypeInt},
		{Ident: ast.TypeString},
	})
	if !ok || got.Ident != ast.TypeUnion || len(got.TypeParams) != 2 {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
}

func TestJoinTypes_singletonOneElementSlice(t *testing.T) {
	t.Parallel()
	got, ok := JoinTypes([]ast.TypeNode{{Ident: ast.TypeBool}})
	if !ok || got.Ident != ast.TypeBool {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
}

func TestJoinTypes_typeParams_allEquivalent(t *testing.T) {
	t.Parallel()
	box := ast.TypeIdent("Box")
	tn := ast.TypeNode{Ident: box, TypeParams: []ast.TypeNode{{Ident: ast.TypeString}}}
	got, ok := JoinTypes([]ast.TypeNode{tn, tn})
	if !ok || got.Ident != box {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
}

func TestJoinTypes_typeParams_heterogeneous(t *testing.T) {
	t.Parallel()
	box := ast.TypeIdent("Box")
	a := ast.TypeNode{Ident: box, TypeParams: []ast.TypeNode{{Ident: ast.TypeString}}}
	b := ast.TypeNode{Ident: box, TypeParams: []ast.TypeNode{{Ident: ast.TypeInt}}}
	got, ok := JoinTypes([]ast.TypeNode{a, b})
	if !ok || got.Ident != ast.TypeUnion || len(got.TypeParams) != 2 {
		t.Fatalf("expected union of Box types, got ok=%v %+v", ok, got)
	}
}

func TestJoinTypesDetailed_outcomes(t *testing.T) {
	t.Parallel()
	t.Run("all_equivalent_two_operands", func(t *testing.T) {
		t.Parallel()
		got, out := JoinTypesDetailed([]ast.TypeNode{
			{Ident: ast.TypeString},
			{Ident: ast.TypeString},
		})
		if out != JoinTypesAllEquivalent || got.Ident != ast.TypeString {
			t.Fatalf("got %+v outcome %v", got, out)
		}
	})
	t.Run("all_equivalent_three_operands", func(t *testing.T) {
		t.Parallel()
		got, out := JoinTypesDetailed([]ast.TypeNode{
			{Ident: ast.TypeString},
			{Ident: ast.TypeString},
			{Ident: ast.TypeString},
		})
		if out != JoinTypesAllEquivalent || got.Ident != ast.TypeString {
			t.Fatalf("got %+v outcome %v", got, out)
		}
	})
	t.Run("singleton_single_operand", func(t *testing.T) {
		t.Parallel()
		got, out := JoinTypesDetailed([]ast.TypeNode{{Ident: ast.TypeString}})
		if out != JoinTypesSingleton || got.Ident != ast.TypeString {
			t.Fatalf("got %+v outcome %v", got, out)
		}
	})
	t.Run("empty_nil", func(t *testing.T) {
		t.Parallel()
		got, out := JoinTypesDetailed(nil)
		if out != JoinTypesEmpty || got.Ident != "" {
			t.Fatalf("got %+v outcome %v", got, out)
		}
	})
	t.Run("empty_non_nil_slice", func(t *testing.T) {
		t.Parallel()
		got, out := JoinTypesDetailed([]ast.TypeNode{})
		if out != JoinTypesEmpty || got.Ident != "" {
			t.Fatalf("got %+v outcome %v", got, out)
		}
	})
	t.Run("heterogeneous_outcome", func(t *testing.T) {
		t.Parallel()
		got, out := JoinTypesDetailed([]ast.TypeNode{
			{Ident: ast.TypeString},
			{Ident: ast.TypeInt},
		})
		if out != JoinTypesHeterogeneous || got.Ident != ast.TypeUnion {
			t.Fatalf("got %+v outcome %v", got, out)
		}
	})
}

func TestDedupeTypesPreservingOrder(t *testing.T) {
	t.Parallel()
	in := []ast.TypeNode{
		{Ident: ast.TypeString},
		{Ident: ast.TypeInt},
		{Ident: ast.TypeString},
	}
	got := DedupeTypesPreservingOrder(in)
	if len(got) != 2 || got[0].Ident != ast.TypeString || got[1].Ident != ast.TypeInt {
		t.Fatalf("DedupeTypesPreservingOrder: %+v", got)
	}
}

func TestDedupeTypesPreservingOrder_table(t *testing.T) {
	t.Parallel()
	box := ast.TypeIdent("Box")
	sBox := ast.TypeNode{Ident: box, TypeParams: []ast.TypeNode{{Ident: ast.TypeString}}}
	iBox := ast.TypeNode{Ident: box, TypeParams: []ast.TypeNode{{Ident: ast.TypeInt}}}
	cases := []struct {
		name string
		in   []ast.TypeNode
		want []ast.TypeIdent
	}{
		{"nil", nil, nil},
		{"empty", []ast.TypeNode{}, nil},
		{"single", []ast.TypeNode{{Ident: ast.TypeString}}, []ast.TypeIdent{ast.TypeString}},
		{"no_duplicates", []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeInt}}, []ast.TypeIdent{ast.TypeString, ast.TypeInt}},
		{
			name: "dedupe_type_params",
			in:   []ast.TypeNode{sBox, iBox, sBox},
			want: []ast.TypeIdent{sBox.Ident, iBox.Ident},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := DedupeTypesPreservingOrder(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("len got %d want %d: %+v", len(got), len(c.want), got)
			}
			for i := range c.want {
				if got[i].Ident != c.want[i] {
					t.Fatalf("[%d] got %s want %s", i, got[i].Ident, c.want[i])
				}
			}
		})
	}
}
