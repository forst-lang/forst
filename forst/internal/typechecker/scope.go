package typechecker

import (
	"forst/internal/ast"
)

// Represents a lexical scope in the program, containing symbols and their definitions
type Scope struct {
	Parent   *Scope
	Node     ast.Node
	Symbols  map[ast.Identifier]Symbol
	Children []*Scope
}

func NewScope(parent *Scope, node ast.Node) *Scope {
	return &Scope{
		Parent:   parent,
		Node:     node,
		Symbols:  make(map[ast.Identifier]Symbol),
		Children: make([]*Scope, 0),
	}
}

func (s *Scope) DefineVariable(name ast.Identifier, typ ast.TypeNode) {
	s.Symbols[name] = Symbol{
		Identifier: name,
		Types:      []ast.TypeNode{typ},
		Kind:       SymbolVariable,
		Scope:      s,
	}
}

// Recursively searches for a variable in the current scope and its ancestors
func (s *Scope) LookupVariable(name ast.Identifier) (Symbol, bool) {
	if symbol, ok := s.Symbols[name]; ok {
		return symbol, true
	}
	if s.Parent != nil {
		return s.Parent.LookupVariable(name)
	}
	return Symbol{}, false
}

func (s *Scope) DefineType(name ast.Identifier, typ ast.TypeNode) {
	s.Symbols[name] = Symbol{
		Identifier: name,
		Types:      []ast.TypeNode{typ},
		Kind:       SymbolType,
		Scope:      s,
	}
}

// Recursively searches for a type definition in the current scope and its ancestors
func (s *Scope) LookupType(name ast.Identifier) (Symbol, bool) {
	if symbol, ok := s.Symbols[name]; ok && symbol.Kind == SymbolType {
		return symbol, true
	}
	if s.Parent != nil {
		return s.Parent.LookupType(name)
	}
	return Symbol{}, false
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

func (s *Scope) IsFunction() bool {
	_, ok := s.Node.(ast.FunctionNode)
	return ok
}
