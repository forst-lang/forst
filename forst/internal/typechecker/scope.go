package typechecker

import (
	"fmt"
	"forst/internal/ast"
	"log"

	"github.com/sirupsen/logrus"
)

// Scope represents a lexical scope in the program, containing symbols and their definitions
type Scope struct {
	Parent   *Scope
	Node     *ast.Node
	Symbols  map[ast.Identifier]Symbol
	Children []*Scope
	log      *logrus.Logger
}

// NewScope creates a new scope
func NewScope(parent *Scope, node *ast.Node, log *logrus.Logger) *Scope {
	return &Scope{
		Parent:   parent,
		Node:     node,
		Symbols:  make(map[ast.Identifier]Symbol),
		Children: make([]*Scope, 0),
		log:      log,
	}
}

// RegisterSymbol registers a symbol in the scope
func (s *Scope) RegisterSymbol(name ast.Identifier, types []ast.TypeNode, kind SymbolKind) {
	s.log.Tracef("[RegisterSymbol] Registering symbol %s with types %v in scope %s", name, types, s.String())
	s.Symbols[name] = Symbol{
		Identifier: name,
		Types:      types,
		Kind:       kind,
		Scope:      s,
	}
}

// LookupVariable recursively searches for a variable in the current scope and its ancestors
func (s *Scope) LookupVariable(name ast.Identifier) (Symbol, bool) {
	if symbol, ok := s.Symbols[name]; ok {
		if symbol.Kind == SymbolVariable {
			return symbol, true
		}
		if symbol.Kind == SymbolParameter {
			if s.IsFunction() || s.IsTypeGuard() {
				return symbol, true
			}
			s.log.WithFields(logrus.Fields{
				"name":     name,
				"scope":    s.String(),
				"kind":     symbol.Kind,
				"function": "LookupVariable",
			}).Debug("Found parameter but parameters are ignored")
		}
	}
	if s.Parent != nil {
		return s.Parent.LookupVariable(name)
	}
	return Symbol{}, false
}

// DefineType defines a type in the scope
func (s *Scope) DefineType(name ast.Identifier, typ ast.TypeNode) {
	s.Symbols[name] = Symbol{
		Identifier: name,
		Types:      []ast.TypeNode{typ},
		Kind:       SymbolType,
		Scope:      s,
	}
}

// LookupType recursively searches for a type definition in the current scope and its ancestors
func (s *Scope) LookupType(name ast.Identifier) (Symbol, bool) {
	if symbol, ok := s.Symbols[name]; ok && symbol.Kind == SymbolType {
		return symbol, true
	}
	if s.Parent != nil {
		return s.Parent.LookupType(name)
	}
	return Symbol{}, false
}

// Symbol represents a symbol in the scope
type Symbol struct {
	Identifier ast.Identifier
	Types      []ast.TypeNode
	Kind       SymbolKind // Variable, Function, Type, etc
	Scope      *Scope     // Where this symbol is defined
	Position   NodePath   // Precise location in AST where symbol is valid
}

// NodePath represents the path from the root to the current node
type NodePath []ast.Node // Path from root to current node

// SymbolKind represents the kind of symbol
type SymbolKind int

const (
	// SymbolVariable represents a variable symbol
	SymbolVariable SymbolKind = iota
	// SymbolFunction represents a function symbol
	SymbolFunction
	// SymbolType represents a type symbol
	SymbolType
	// SymbolParameter represents a parameter symbol
	SymbolParameter
	// SymbolStruct represents a struct symbol
	SymbolStruct
	// SymbolEnum represents an enum symbol
	SymbolEnum
	// SymbolTypeGuard represents a type guard symbol
	SymbolTypeGuard
)

// IsFunction checks if the scope is a function
func (s *Scope) IsFunction() bool {
	if s.Node == nil {
		panic("Cannot call IsFunction on global scope")
	}
	_, ok := (*s.Node).(ast.FunctionNode)
	return ok
}

// IsTypeGuard checks if the scope is a type guard
func (s *Scope) IsTypeGuard() bool {
	if s.Node == nil {
		panic("Cannot call IsTypeGuard on global scope")
	}
	_, ok := (*s.Node).(ast.TypeGuardNode)
	if !ok {
		_, ok = (*s.Node).(*ast.TypeGuardNode)
	}
	return ok
}

// LookupVariableType looks up a variable's type in the current scope
func (s *Scope) LookupVariableType(name ast.Identifier) ([]ast.TypeNode, bool) {
	// Start from the current scope and work up
	for i := len(s.Symbols) - 1; i >= 0; i-- {
		if sym, exists := s.Symbols[name]; exists {
			return sym.Types, true
		}
	}
	return nil, false
}

func (s *Scope) String() string {
	if s.IsGlobal() {
		return "GlobalScope"
	}
	if s.Node == nil {
		log.Fatalf("Scope node is nil in non-global scope")
	}
	return fmt.Sprintf("Scope(%v)", (*s.Node).String())
}

func (s *Scope) IsGlobal() bool {
	return s.Parent == nil
}
