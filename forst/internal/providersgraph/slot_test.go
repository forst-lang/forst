package providersgraph

import (
	"testing"

	"forst/internal/ast"
)

func TestOrderSlots_dedupesAndSortsByKey(t *testing.T) {
	t.Parallel()
	slots := []Slot{
		{Key: "Clock", RootIdent: "Clock"},
		{Key: "Logger", RootIdent: "Logger"},
		{Key: "Logger", RootIdent: "LoggerDup"},
	}
	got := OrderSlots(slots)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 unique keys", len(got))
	}
	if got[0].Key != "Clock" || got[1].Key != "Logger" {
		t.Fatalf("order = %v", []string{got[0].Key, got[1].Key})
	}
}

func TestAddSlotToFunction_skipsEmptyKeyAndDuplicates(t *testing.T) {
	t.Parallel()
	m := make(map[ast.Identifier][]Slot)
	if AddSlotToFunction(m, "f", Slot{}) {
		t.Fatal("empty key should not add")
	}
	if !AddSlotToFunction(m, "f", Slot{Key: "Logger", RootIdent: "Logger"}) {
		t.Fatal("first add should succeed")
	}
	if AddSlotToFunction(m, "f", Slot{Key: "Logger", RootIdent: "Other"}) {
		t.Fatal("duplicate key should not add")
	}
	if len(m["f"]) != 1 {
		t.Fatalf("slots = %v", m["f"])
	}
}

func TestSlotsFromDirectMap_emptyReturnsNil(t *testing.T) {
	t.Parallel()
	if got := SlotsFromDirectMap(nil); got != nil {
		t.Fatalf("got %#v", got)
	}
}

func TestRootIdentsFromSlots_emptyReturnsNil(t *testing.T) {
	t.Parallel()
	if got := RootIdentsFromSlots(nil); got != nil {
		t.Fatalf("got %#v", got)
	}
}

func TestRootIdentsFromSlots_returnsOrderedIdents(t *testing.T) {
	t.Parallel()
	got := RootIdentsFromSlots([]Slot{
		{RootIdent: "Logger"},
		{RootIdent: "Clock"},
	})
	if len(got) != 2 || got[0] != "Logger" || got[1] != "Clock" {
		t.Fatalf("got %#v", got)
	}
}

func TestOrderSlots_singleElementPassthrough(t *testing.T) {
	t.Parallel()
	in := []Slot{{Key: "Only", RootIdent: "Only"}}
	got := OrderSlots(in)
	if len(got) != 1 || got[0].Key != "Only" {
		t.Fatalf("got %#v", got)
	}
}

func TestSlotsFromDirectMap_ordersKeys(t *testing.T) {
	t.Parallel()
	got := SlotsFromDirectMap(map[string]Slot{
		"B": {Key: "B", RootIdent: "B"},
		"A": {Key: "A", RootIdent: "A"},
	})
	if len(got) != 2 || got[0].Key != "A" || got[1].Key != "B" {
		t.Fatalf("got %#v", got)
	}
}

func TestProviderScopeKeyPresent(t *testing.T) {
	t.Parallel()
	slot := Slot{RootIdent: "Logger"}
	scope := map[string]ast.TypeNode{"Logger": {Ident: "Logger"}}
	if !ProviderScopeKeyPresent(slot, scope) {
		t.Fatal("Logger should be present")
	}
	if ProviderScopeKeyPresent(slot, nil) {
		t.Fatal("nil scope should be absent")
	}
	if ProviderScopeKeyPresent(Slot{RootIdent: "Clock"}, scope) {
		t.Fatal("Clock should be absent")
	}
}
