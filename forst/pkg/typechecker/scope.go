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

// ScopeStack manages the stack of scopes during type checking
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
