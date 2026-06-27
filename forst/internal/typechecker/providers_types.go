package typechecker

import (
	"forst/internal/ast"
	"forst/internal/providersgraph"
)

// ProviderSlot is one inferred Provider requirement for a function (root ident + contract type).
type ProviderSlot = providersgraph.Slot

// ProviderScope holds merged Provider wiring keys in scope during a with-block.
type ProviderScope struct {
	keys     map[string]ast.TypeNode
	shadowed map[string]bool
}

type callSiteRecord struct {
	Callee      ast.Identifier
	ScopeKeys map[string]ast.TypeNode
	Span        ast.SourceSpan
}

type pendingWithCheck struct {
	with      ast.WithNode
	innerKeys map[string]struct{}
}

func newProviderScope() ProviderScope {
	return ProviderScope{
		keys:     make(map[string]ast.TypeNode),
		shadowed: make(map[string]bool),
	}
}
