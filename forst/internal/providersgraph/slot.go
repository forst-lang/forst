package providersgraph

import (
	"sort"

	"forst/internal/ast"
)

// Slot is one inferred Provider requirement for a function (root ident + contract type).
type Slot struct {
	RootIdent    ast.Identifier
	ContractType ast.TypeNode
	Key          string
	// SourcePkg is the Forst package that defined this slot (set when propagated cross-package).
	SourcePkg string
}

// ProviderScopeSnapshot is merged wiring keys in scope at a call site.
type ProviderScopeSnapshot map[string]ast.TypeNode

// OrderSlots returns slots sorted by Key.
func OrderSlots(slots []Slot) []Slot {
	if len(slots) <= 1 {
		return slots
	}
	seen := make(map[string]Slot, len(slots))
	for _, s := range slots {
		seen[s.Key] = s
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Slot, 0, len(keys))
	for _, k := range keys {
		out = append(out, seen[k])
	}
	return out
}

// AddSlotToFunction appends slot to fn when Key is new.
func AddSlotToFunction(m map[ast.Identifier][]Slot, fn ast.Identifier, slot Slot) bool {
	if slot.Key == "" {
		return false
	}
	existing := m[fn]
	for _, s := range existing {
		if s.Key == slot.Key {
			return false
		}
	}
	m[fn] = append(existing, slot)
	return true
}

// SlotsFromDirectMap converts a per-key direct slot map to an ordered slice.
func SlotsFromDirectMap(m map[string]Slot) []Slot {
	if len(m) == 0 {
		return nil
	}
	out := make([]Slot, 0, len(m))
	for _, slot := range m {
		out = append(out, slot)
	}
	return OrderSlots(out)
}

// RootIdentsFromSlots returns ordered root contract idents for discovery JSON / LSP.
func RootIdentsFromSlots(slots []Slot) []string {
	if len(slots) == 0 {
		return nil
	}
	out := make([]string, len(slots))
	for i, s := range slots {
		out[i] = string(s.RootIdent)
	}
	return out
}

// ProviderScopeKeyPresent reports whether scope supplies the slot's root contract key.
func ProviderScopeKeyPresent(slot Slot, scope map[string]ast.TypeNode) bool {
	if scope == nil {
		return false
	}
	_, ok := scope[string(slot.RootIdent)]
	return ok
}
