package typechecker

import (
	"forst/pkg/ast"
)

// ScopeStack manages a stack of scopes during type checking
type ScopeStack struct {
	scopes  map[NodeHash]*Scope
	current *Scope
	Hasher  *StructuralHasher
}

// NewScopeStack creates a new scope stack with a global scope
func NewScopeStack(hasher *StructuralHasher) *ScopeStack {
	globalScope := &Scope{
		Symbols: make(map[ast.Identifier]Symbol),
	}
	return &ScopeStack{
		scopes:  make(map[NodeHash]*Scope),
		current: globalScope,
		Hasher:  hasher,
	}
}

// PushScope pushes a new scope onto the stack
func (ss *ScopeStack) PushScope(node ast.Node) {
	hash := ss.Hasher.HashNode(node)
	scope := &Scope{
		Parent:   ss.current,
		Node:     node,
		Symbols:  make(map[ast.Identifier]Symbol),
		Children: make([]*Scope, 0),
	}
	ss.current.Children = append(ss.current.Children, scope)
	ss.current = scope
	ss.scopes[hash] = scope
}

// PopScope pops the current scope from the stack
func (ss *ScopeStack) PopScope() {
	if ss.current.Parent != nil {
		ss.current = ss.current.Parent
	}
}

// CurrentScope returns the current scope
func (ss *ScopeStack) CurrentScope() *Scope {
	return ss.current
}

// FindScope finds a scope by its node
func (ss *ScopeStack) FindScope(node ast.Node) *Scope {
	hash := ss.Hasher.HashNode(node)
	return ss.scopes[hash]
}

// GlobalScope returns the global scope (root scope)
func (ss *ScopeStack) GlobalScope() *Scope {
	scope := ss.current
	for scope.Parent != nil {
		scope = scope.Parent
	}
	return scope
}
