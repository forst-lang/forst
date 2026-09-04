package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestSymbolID_shadowingDistinct(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	h := New(log, false).Hasher
	ss := NewScopeStack(h, log)
	outer := ss.currentScope()
	outer.RegisterSymbol("x", []ast.TypeNode{{Ident: ast.TypeInt}}, SymbolVariable)
	outerID, ok := outer.LookupSymbolID("x")
	if !ok || outerID == 0 {
		t.Fatalf("outer id: ok=%v id=%d", ok, outerID)
	}
	fn := ast.Node(ast.FunctionNode{Ident: ast.Ident{ID: "f"}})
	inner := ss.pushScope(fn)
	inner.RegisterSymbol("x", []ast.TypeNode{{Ident: ast.TypeString}}, SymbolVariable)
	innerID, ok := inner.LookupSymbolID("x")
	if !ok || innerID == 0 {
		t.Fatalf("inner id: ok=%v id=%d", ok, innerID)
	}
	if outerID == innerID {
		t.Fatalf("shadowed binding must get a new SymbolID; both %d", outerID)
	}
	// Outer lookup from outer scope still sees outer ID.
	if got, _ := outer.LookupSymbolID("x"); got != outerID {
		t.Fatalf("outer still %d, got %d", outerID, got)
	}
}

func TestSymbolRef_remapStable(t *testing.T) {
	refs := []SymbolRef{
		{Package: "b", Name: "y"},
		{Package: "a", Name: "x"},
		{Package: "a", Name: "x"}, // dedupe
	}
	byRef, byID := RemapSymbolRefs(refs)
	if len(byRef) != 2 || len(byID) != 2 {
		t.Fatalf("want 2 unique refs, got byRef=%d byID=%d", len(byRef), len(byID))
	}
	idA := byRef[SymbolRef{Package: "a", Name: "x"}]
	idB := byRef[SymbolRef{Package: "b", Name: "y"}]
	if idA == 0 || idB == 0 || idA == idB {
		t.Fatalf("ids a=%d b=%d", idA, idB)
	}
	// Remap again yields same dense assignment order (by Key sort).
	byRef2, _ := RemapSymbolRefs(refs)
	if byRef2[SymbolRef{Package: "a", Name: "x"}] != idA {
		t.Fatal("remap not stable")
	}
}

func TestAccessPath_fieldIndexDeref(t *testing.T) {
	p := AccessPath{
		Root: 1,
		Steps: []AccessStep{
			{Kind: AccessField, Field: "addr"},
			{Kind: AccessField, Field: "street"},
			{Kind: AccessIndexAny},
			{Kind: AccessDeref},
		},
	}
	key := p.PathKey()
	if key != "1.f:addr.f:street[*].*" {
		t.Fatalf("PathKey: %q", key)
	}
}

func TestAccessPath_interning(t *testing.T) {
	pi := NewPathInterner()
	a := pi.Intern(AccessPath{Root: 3, Steps: []AccessStep{{Kind: AccessField, Field: "age"}}})
	b := pi.Intern(AccessPath{Root: 3, Steps: []AccessStep{{Kind: AccessField, Field: "age"}}})
	c := pi.Intern(AccessPath{Root: 3, Steps: []AccessStep{{Kind: AccessField, Field: "name"}}})
	if a != b {
		t.Fatal("identical paths must intern to same pointer")
	}
	if a == c {
		t.Fatal("different paths must not share pointer")
	}
	if pi.Len() != 2 {
		t.Fatalf("Len=%d", pi.Len())
	}
}
