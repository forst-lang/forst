package typechecker

import (
	"forst/internal/ast"
	"forst/internal/usablesgraph"
)

// UsableSlot is one inferred Usable requirement for a function (root ident + contract type).
type UsableSlot = usablesgraph.Slot

// Ambient holds merged Usable wiring keys in scope during a with-block.
type Ambient struct {
	keys     map[string]ast.TypeNode
	shadowed map[string]bool
}

type callSiteRecord struct {
	Callee      ast.Identifier
	AmbientKeys map[string]ast.TypeNode
	Span        ast.SourceSpan
}

type pendingWithCheck struct {
	with      ast.WithNode
	innerKeys map[string]struct{}
}

func newAmbient() Ambient {
	return Ambient{
		keys:     make(map[string]ast.TypeNode),
		shadowed: make(map[string]bool),
	}
}
