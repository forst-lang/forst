package typechecker

// SymbolID is a local analysis identity for a binding. Serialized metadata uses SymbolRef
// and remaps on load; source strings are diagnostic-only.
type SymbolID uint32

// SymbolRef is the cross-package / serialized form of a binding identity.
type SymbolRef struct {
	Package string
	Name    string
}

// Key returns a stable map key for interning and remapping.
func (r SymbolRef) Key() string {
	return r.Package + "\x00" + r.Name
}

// RemapSymbolRefs rebuilds a SymbolID table from qualified refs (load / cross-package).
// Local IDs are assigned densely starting at 1 in stable Key() order of refs.
func RemapSymbolRefs(refs []SymbolRef) (map[SymbolRef]SymbolID, map[SymbolID]SymbolRef) {
	byRef := make(map[SymbolRef]SymbolID, len(refs))
	byID := make(map[SymbolID]SymbolRef, len(refs))
	var next SymbolID = 1
	seen := make(map[string]SymbolRef, len(refs))
	keys := make([]string, 0, len(refs))
	for _, r := range refs {
		k := r.Key()
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = r
		keys = append(keys, k)
	}
	sortStrings(keys)
	for _, k := range keys {
		r := seen[k]
		id := next
		next++
		byRef[r] = id
		byID[id] = r
	}
	return byRef, byID
}

func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j-1] > ss[j]; j-- {
			ss[j-1], ss[j] = ss[j], ss[j-1]
		}
	}
}
