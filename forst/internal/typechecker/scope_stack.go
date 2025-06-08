package typechecker

import (
	"fmt"
	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

// ScopeStack manages a stack of scopes during type checking
type ScopeStack struct {
	scopes  map[NodeHash]*Scope
	current *Scope
	Hasher  *StructuralHasher
	log     *logrus.Logger
}

// NewScopeStack creates a new stack with a global scope
func NewScopeStack(hasher *StructuralHasher, log *logrus.Logger) *ScopeStack {
	globalScope := NewScope(nil, nil, log)
	return &ScopeStack{
		scopes:  make(map[NodeHash]*Scope),
		current: globalScope,
		Hasher:  hasher,
		log:     log,
	}
}

// pushScope creates and pushes a new scope for the given AST node
func (ss *ScopeStack) pushScope(node ast.Node) *Scope {
	hash := ss.Hasher.HashNode(node)
	scope := NewScope(ss.current, &node, ss.log)
	ss.current.Children = append(ss.current.Children, scope)
	ss.current = scope
	ss.scopes[hash] = scope
	return scope
}

// popScope returns to the parent scope if one exists
func (ss *ScopeStack) popScope() {
	if ss.current.Parent != nil {
		ss.current = ss.current.Parent
	}
}

// currentScope returns the current scope
func (ss *ScopeStack) currentScope() *Scope {
	return ss.current
}

// restoreScope restores a scope by its AST node
func (ss *ScopeStack) restoreScope(node ast.Node) error {
	scope, exists := ss.findScope(node)
	if !exists {
		return fmt.Errorf("scope not found for node %s", node.String())
	}
	ss.current = scope
	return nil
}

// findScope looks up a scope by its AST node
func (ss *ScopeStack) findScope(node ast.Node) (*Scope, bool) {
	hash := ss.Hasher.HashNode(node)
	scope, exists := ss.scopes[hash]
	return scope, exists
}

// globalScope returns the root scope
func (ss *ScopeStack) globalScope() *Scope {
	scope := ss.current
	for scope.Parent != nil {
		scope = scope.Parent
	}
	return scope
}

// LookupVariableType looks up a variable's type in the current scope stack
func (ss *ScopeStack) LookupVariableType(name ast.Identifier) ([]ast.TypeNode, bool) {
	// Start from the current scope and work up through parents
	scope := ss.current
	for scope != nil {
		if types, exists := scope.LookupVariableType(name); exists {
			return types, true
		}
		scope = scope.Parent
	}
	return nil, false
}
