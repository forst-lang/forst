package typechecker

import (
	"sort"
	"strings"

	"forst/internal/ast"
)

// EffectiveAmbientKeys merges wiring from nested with-blocks and returns sorted root keys available in that scope.
func (tc *TypeChecker) EffectiveAmbientKeys(chain []ast.WithNode) ([]string, error) {
	labels, err := tc.EffectiveAmbientKeyLabels(chain)
	if err != nil {
		return nil, err
	}
	keys := make([]string, len(labels))
	for i, l := range labels {
		keys[i] = l.Key
	}
	return keys, nil
}

// EffectiveAmbientKeyLabel is one merged ambient contract root with optional shadow marker (inner overlays outer).
type EffectiveAmbientKeyLabel struct {
	Key      string
	Shadowed bool
}

// EffectiveAmbientKeyLabels merges nested with-blocks and returns sorted labels for LSP ambient hover.
func (tc *TypeChecker) EffectiveAmbientKeyLabels(chain []ast.WithNode) ([]EffectiveAmbientKeyLabel, error) {
	if len(chain) == 0 {
		return nil, nil
	}
	var merged Ambient
	shadowedKeys := make(map[string]bool)
	for i, w := range chain {
		amb, err := tc.ambientFromWiringExpr(w.Wiring, w.Span)
		if err != nil {
			return nil, err
		}
		if i > 0 {
			for k := range amb.keys {
				if _, had := merged.keys[k]; had {
					shadowedKeys[k] = true
				}
			}
		}
		merged = tc.mergeAmbient(merged, amb)
	}
	keys := make([]string, 0, len(merged.keys))
	for k := range merged.keys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	labels := make([]EffectiveAmbientKeyLabel, len(keys))
	for i, k := range keys {
		labels[i] = EffectiveAmbientKeyLabel{
			Key:      k,
			Shadowed: shadowedKeys[k],
		}
	}
	return labels, nil
}

// KnownUsableRootKeys returns sorted contract root identifiers valid in with wiring maps.
func (tc *TypeChecker) KnownUsableRootKeys() []string {
	if len(tc.knownUsableRoots) == 0 {
		return nil
	}
	keys := make([]string, 0, len(tc.knownUsableRoots))
	for k := range tc.knownUsableRoots {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// UsableImplTypeNamesForContract returns type names (and &Type) assignable to a wiring key.
func (tc *TypeChecker) UsableImplTypeNamesForContract(contractRoot string) []string {
	contract, ok := tc.knownUsableRoots[contractRoot]
	if !ok {
		return nil
	}
	var out []string
	for ident, def := range tc.Defs {
		if _, ok := def.(ast.TypeDefNode); !ok {
			continue
		}
		impl := ast.TypeNode{Ident: ident}
		if tc.wiringValueAssignable(impl, contract) {
			out = append(out, string(ident))
		}
		ptr := ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{impl}}
		if tc.wiringValueAssignable(ptr, contract) {
			out = append(out, "&"+string(ident))
		}
	}
	sort.Strings(out)
	return out
}

func (tc *TypeChecker) ambientSatisfiesAllSlots(ambient map[string]ast.TypeNode, slots []UsableSlot) bool {
	for _, slot := range slots {
		if !tc.ambientSatisfiesSlot(slot, ambient) {
			return false
		}
	}
	return true
}

func (tc *TypeChecker) validateAllCallSites() error {
	for caller, sites := range tc.functionCallSites {
		for _, site := range sites {
			if err := tc.validateCallSite(caller, site); err != nil {
				return err
			}
		}
	}
	return nil
}

func (tc *TypeChecker) isWiringRoot(fn ast.FunctionNode) bool {
	return ast.IsUsablesWiringRoot(fn.Ident.ID, ast.ParamTypesFromFunction(fn))
}

func obligationChain(root ast.Identifier, slots []UsableSlot) string {
	parts := []string{string(root)}
	seen := map[string]struct{}{string(root): {}}
	for _, slot := range slots {
		name := string(slot.RootIdent)
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		parts = append(parts, name)
	}
	return strings.Join(parts, " → ")
}

// UsablesObligationChain formats root → …usable roots for diagnostics and LSP.
func UsablesObligationChain(root ast.Identifier, slots []UsableSlot) string {
	return obligationChain(root, slots)
}

// CallSiteObligationChain formats caller → callee → …usable roots at a call site.
func CallSiteObligationChain(caller, callee ast.Identifier, slots []UsableSlot) string {
	parts := []string{string(caller), string(callee)}
	seen := map[string]struct{}{string(caller): {}, string(callee): {}}
	for _, slot := range slots {
		name := string(slot.RootIdent)
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		parts = append(parts, name)
	}
	return strings.Join(parts, " → ")
}
