package typechecker

import (
	"forst/pkg/ast"
	"slices"

	log "github.com/sirupsen/logrus"
)

type Scope struct {
	Parent   *Scope
	Node     ast.Node                  // The AST node that created this scope (function, block, if, etc)
	Symbols  map[ast.Identifier]Symbol // All symbols defined in this scope
	Children []*Scope                  // Child scopes (blocks, branches)
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

// Add these methods to manage symbols
func (tc *TypeChecker) storeSymbol(ident ast.Identifier, typ []ast.TypeNode, kind SymbolKind) {
	symbol := Symbol{
		Identifier: ident,
		Types:      typ,
		Kind:       kind,
		Scope:      tc.currentScope,
		Position:   slices.Clone(tc.path), // Copy current path
	}
	tc.currentScope.Symbols[ident] = symbol
}

func (tc *TypeChecker) FindScope(node ast.Node) *Scope {
	hash := tc.Hasher.Hash(node)
	return tc.scopes[hash]
}

func (tc *TypeChecker) pushScope(node ast.Node) {
	log.Tracef("pushScope %v (hash: %d)\n", node, tc.Hasher.Hash(node))
	newScope := &Scope{
		Parent:   tc.currentScope,
		Node:     node,
		Symbols:  make(map[ast.Identifier]Symbol),
		Children: make([]*Scope, 0),
	}

	if tc.currentScope != nil {
		tc.currentScope.Children = append(tc.currentScope.Children, newScope)
	}
	hash := tc.Hasher.Hash(node)
	tc.scopes[hash] = newScope
	tc.currentScope = newScope
}

func (tc *TypeChecker) popScope() {
	log.Trace("popScope")
	if tc.currentScope.Parent != nil {
		tc.currentScope = tc.currentScope.Parent
	} else {
		panic("no parent scope to pop from, we are already in the global scope")
	}
}
