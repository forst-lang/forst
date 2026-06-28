package typechecker

import (
	"sort"
	"strings"

	"forst/internal/ast"
)

// EffectiveScopeKeys merges wiring from nested with-blocks and returns sorted root keys available in that scope.
func (tc *TypeChecker) EffectiveScopeKeys(chain []ast.WithNode) ([]string, error) {
	labels, err := tc.EffectiveScopeKeyLabels(chain)
	if err != nil {
		return nil, err
	}
	keys := make([]string, len(labels))
	for i, l := range labels {
		keys[i] = l.Key
	}
	return keys, nil
}

// EffectiveScopeKeyLabel is one merged scope contract root with optional shadow marker (inner overlays outer).
type EffectiveScopeKeyLabel struct {
	Key      string
	Shadowed bool
}

// EffectiveScopeKeyLabels merges nested with-blocks and returns sorted labels for LSP scope hover.
func (tc *TypeChecker) EffectiveScopeKeyLabels(chain []ast.WithNode) ([]EffectiveScopeKeyLabel, error) {
	if len(chain) == 0 {
		return nil, nil
	}
	var merged ProviderScope
	shadowedKeys := make(map[string]bool)
	for i, w := range chain {
		amb, err := tc.ambientFromWiringExpr(w.Wiring, w.Span)
		if err != nil {
			return nil, err
		}
		if i > 0 {
			for k := range amb.Keys {
				if _, had := merged.Keys[k]; had {
					shadowedKeys[k] = true
				}
			}
		}
		merged = tc.mergeProviderScope(merged, amb)
	}
	keys := make([]string, 0, len(merged.Keys))
	for k := range merged.Keys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	labels := make([]EffectiveScopeKeyLabel, len(keys))
	for i, k := range keys {
		labels[i] = EffectiveScopeKeyLabel{
			Key:      k,
			Shadowed: shadowedKeys[k],
		}
	}
	return labels, nil
}

// KnownProviderRootKeys returns sorted contract root identifiers valid in with wiring maps.
func (tc *TypeChecker) KnownProviderRootKeys() []string {
	eng := tc.providersEngine()
	if len(eng.KnownRoots) == 0 {
		return nil
	}
	keys := make([]string, 0, len(eng.KnownRoots))
	for k := range eng.KnownRoots {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ProviderImplTypeNamesForContract returns type names (and &Type) assignable to a wiring key.
func (tc *TypeChecker) ProviderImplTypeNamesForContract(contractRoot string) []string {
	contract, ok := tc.providersEngine().KnownRoots[contractRoot]
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

func (tc *TypeChecker) scopeSatisfiesAllSlots(scope map[string]ast.TypeNode, slots []ProviderSlot) bool {
	for _, slot := range slots {
		if !tc.scopeSatisfiesSlot(slot, scope) {
			return false
		}
	}
	return true
}

func (tc *TypeChecker) isWiringRoot(fn ast.FunctionNode) bool {
	return ast.IsProvidersWiringRoot(fn.Ident.ID, ast.ParamTypesFromFunction(fn))
}

func obligationChain(root ast.Identifier, slots []ProviderSlot) string {
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

// ProvidersObligationChain formats root → …provider roots for diagnostics and LSP.
func ProvidersObligationChain(root ast.Identifier, slots []ProviderSlot) string {
	return obligationChain(root, slots)
}

// CallSiteObligationChain formats caller → callee → …provider roots at a call site.
func CallSiteObligationChain(caller, callee ast.Identifier, slots []ProviderSlot) string {
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
