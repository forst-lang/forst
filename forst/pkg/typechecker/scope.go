package typechecker

import (
	"forst/pkg/ast"
)

// Scope represents a lexical scope in the program
type Scope struct {
	Parent   *Scope
	Node     ast.Node
	Symbols  map[ast.Identifier]Symbol
	Children []*Scope
}

// NewScope creates a new scope
func NewScope(parent *Scope, node ast.Node) *Scope {
	return &Scope{
		Parent:   parent,
		Node:     node,
		Symbols:  make(map[ast.Identifier]Symbol),
		Children: make([]*Scope, 0),
	}
}

// DefineVariable defines a variable in the current scope
func (s *Scope) DefineVariable(name ast.Identifier, typ ast.TypeNode) {
	s.Symbols[name] = Symbol{
		Identifier: name,
		Types:      []ast.TypeNode{typ},
		Kind:       SymbolVariable,
		Scope:      s,
	}
}

// LookupVariable looks up a variable in the current scope and its parents
func (s *Scope) LookupVariable(name ast.Identifier) (Symbol, bool) {
	if symbol, ok := s.Symbols[name]; ok {
		return symbol, true
	}
	if s.Parent != nil {
		return s.Parent.LookupVariable(name)
	}
	return Symbol{}, false
}

// DefineType defines a type in the current scope
func (s *Scope) DefineType(name ast.Identifier, typ ast.TypeNode) {
	s.Symbols[name] = Symbol{
		Identifier: name,
		Types:      []ast.TypeNode{typ},
		Kind:       SymbolType,
		Scope:      s,
	}
}

// LookupType looks up a type in the current scope and its parents
func (s *Scope) LookupType(name ast.Identifier) (Symbol, bool) {
	if symbol, ok := s.Symbols[name]; ok && symbol.Kind == SymbolType {
		return symbol, true
	}
	if s.Parent != nil {
		return s.Parent.LookupType(name)
	}
	return Symbol{}, false
}

// ScopeManager manages the scope stack for type checking
type ScopeManager struct {
	current *Scope
	scopes  map[NodeHash]*Scope
	Hasher  *StructuralHasher
}

// NewScopeManager creates a new scope manager
func NewScopeManager() *ScopeManager {
	return &ScopeManager{
		current: NewScope(nil, nil),
		scopes:  make(map[NodeHash]*Scope),
		Hasher:  NewStructuralHasher(),
	}
}

// PushScope pushes a new scope onto the stack
func (sm *ScopeManager) PushScope(node ast.Node) {
	newScope := NewScope(sm.current, node)
	sm.current.Children = append(sm.current.Children, newScope)
	if node != nil {
		hash := sm.Hasher.HashNode(node)
		sm.scopes[hash] = newScope
	}
	sm.current = newScope
}

// PopScope pops the current scope from the stack
func (sm *ScopeManager) PopScope() {
	if sm.current.Parent != nil {
		sm.current = sm.current.Parent
	}
}

// CurrentScope returns the current scope
func (sm *ScopeManager) CurrentScope() *Scope {
	return sm.current
}

// FindScope finds a scope by its node
func (sm *ScopeManager) FindScope(node ast.Node) *Scope {
	hash := sm.Hasher.HashNode(node)
	return sm.scopes[hash]
}

// GlobalScope returns the global scope (root scope)
func (sm *ScopeManager) GlobalScope() *Scope {
	scope := sm.current
	for scope.Parent != nil {
		scope = scope.Parent
	}
	return scope
}

type Symbol struct {
	Identifier ast.Identifier
	Types      []ast.TypeNode
	Kind       SymbolKind // Variable, Function, Type, etc
	Scope      *Scope     // Where this symbol is defined
	Position   NodePath   // Precise location in AST where symbol is valid
}

type NodePath []ast.Node // Path from root to current node

type SymbolKind int

const (
	SymbolVariable SymbolKind = iota
	SymbolFunction
	SymbolType
	SymbolParameter
)

// IsFunction returns true if this scope is for a function
func (s *Scope) IsFunction() bool {
	_, ok := s.Node.(ast.FunctionNode)
	return ok
}
