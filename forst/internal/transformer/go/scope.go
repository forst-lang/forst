package transformergo

import (
	"forst/internal/ast"
	"forst/internal/typechecker"
)

// restoreScope restores the scope for a given node
func (t *Transformer) restoreScope(node ast.Node) error {
	return t.TypeChecker.RestoreScope(node)
}

// currentScope returns the current scope from the type checker
func (t *Transformer) currentScope() *typechecker.Scope {
	return t.TypeChecker.CurrentScope()
}
