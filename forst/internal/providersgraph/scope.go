package providersgraph

import "forst/internal/ast"

// ProviderScope holds merged Provider wiring keys in scope during a with-block.
type ProviderScope struct {
	Keys     map[string]ast.TypeNode
	Shadowed map[string]bool
}

// NewProviderScope returns an empty wiring scope.
func NewProviderScope() ProviderScope {
	return ProviderScope{
		Keys:     make(map[string]ast.TypeNode),
		Shadowed: make(map[string]bool),
	}
}

// Snapshot returns a flat map of merged scope keys for call-site satisfaction checks.
func (s ProviderScope) Snapshot() ProviderScopeSnapshot {
	if len(s.Keys) == 0 {
		return nil
	}
	out := make(ProviderScopeSnapshot, len(s.Keys))
	for k, v := range s.Keys {
		out[k] = v
	}
	return out
}

// MergeProviderScope combines outer and inner with-block scopes (inner shadows outer keys).
func MergeProviderScope(outer, inner ProviderScope) ProviderScope {
	out := NewProviderScope()
	for k, v := range outer.Keys {
		if inner.Shadowed[k] {
			continue
		}
		out.Keys[k] = v
	}
	for k, v := range inner.Keys {
		out.Keys[k] = v
	}
	for k := range inner.Shadowed {
		out.Shadowed[k] = true
	}
	return out
}

// MergeScopeStack merges a stack of with-block scopes outer-to-inner.
func MergeScopeStack(stack []ProviderScope) ProviderScopeSnapshot {
	if len(stack) == 0 {
		return nil
	}
	merged := stack[0]
	for i := 1; i < len(stack); i++ {
		merged = MergeProviderScope(merged, stack[i])
	}
	return merged.Snapshot()
}
