package transformergo

import (
	"forst/internal/ast"
)

// pushScope creates a new scope for the given node
func (t *Transformer) pushScope(node ast.Node) {
	t.currentScope = t.TypeChecker.PushScope(node)
}

// popScope removes the current scope and returns to the parent scope
func (t *Transformer) popScope() {
	t.TypeChecker.PopScope()
	t.currentScope = t.TypeChecker.CurrentScope()
}
