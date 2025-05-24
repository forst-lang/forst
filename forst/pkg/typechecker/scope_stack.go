package typechecker

import (
	"forst/pkg/ast"
)

// Manages a stack of scopes during type checking
type ScopeStack struct {
	scopes  map[NodeHash]*Scope
	current *Scope
	Hasher  *StructuralHasher
}

// Creates a new stack with a global scope
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

// Creates and pushes a new scope for the given AST node
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

// Returns to the parent scope if one exists
func (ss *ScopeStack) PopScope() {
	if ss.current.Parent != nil {
		ss.current = ss.current.Parent
	}
}

func (ss *ScopeStack) CurrentScope() *Scope {
	return ss.current
}

// Looks up a scope by its AST node
func (ss *ScopeStack) FindScope(node ast.Node) *Scope {
	hash := ss.Hasher.HashNode(node)
	return ss.scopes[hash]
}

// Returns the root scope
func (ss *ScopeStack) GlobalScope() *Scope {
	scope := ss.current
	for scope.Parent != nil {
		scope = scope.Parent
	}
	return scope
}
