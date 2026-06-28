package typechecker

import (
	"forst/internal/ast"
	"forst/internal/providersgraph"
)

// ProviderSlot is one inferred Provider requirement for a function (root ident + contract type).
type ProviderSlot = providersgraph.Slot

type pendingWithCheck struct {
	with      ast.WithNode
	innerKeys map[string]struct{}
}

// ProviderScope is an alias for the canonical scope type in providersgraph.
type ProviderScope = providersgraph.ProviderScope

func newProviderScope() ProviderScope {
	return providersgraph.NewProviderScope()
}
