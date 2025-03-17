package typechecker

import (
	"forst/pkg/ast"

	log "github.com/sirupsen/logrus"
)

type TypeChecker struct {
	// Use value-based Ident keys instead of pointers
	Types        map[NodeHash][]ast.TypeNode
	Defs         map[ast.Identifier]ast.Node
	Uses         map[ast.Identifier][]ast.Node
	Functions    map[ast.Identifier]FunctionSignature
	Hasher       *StructuralHasher
	currentScope *Scope
	globalScope  *Scope
	scopes       map[NodeHash]*Scope // Map AST nodes to their scopes
	path         NodePath            // Track current position in AST
}

func New() *TypeChecker {
	globalScope := &Scope{
		Parent:  nil,
		Symbols: make(map[ast.Identifier]Symbol),
	}

	return &TypeChecker{
		Types:        make(map[NodeHash][]ast.TypeNode),
		Defs:         make(map[ast.Identifier]ast.Node),
		Uses:         make(map[ast.Identifier][]ast.Node),
		Functions:    make(map[ast.Identifier]FunctionSignature),
		Hasher:       &StructuralHasher{},
		currentScope: globalScope,
		globalScope:  globalScope,
		scopes:       make(map[NodeHash]*Scope),
		path:         make(NodePath, 0),
	}
}

// First pass: collect all type information
func (tc *TypeChecker) CollectTypes(nodes []ast.Node) error {
	for _, node := range nodes {
		switch n := node.(type) {
		case ast.FunctionNode:
			tc.registerFunction(n)
			// case ast.TypeDeclarationNode:
			// 	tc.registerType(n)
		}
	}
	return nil
}

// CheckTypes performs type inference and collects type information
func (tc *TypeChecker) CheckTypes(nodes []ast.Node) error {
	// First pass: collect function signatures and explicit types
	for _, node := range nodes {
		tc.path = append(tc.path, node)
		if err := tc.collectExplicitTypes(node); err != nil {
			return err
		}
		tc.path = tc.path[:len(tc.path)-1]
	}

	// Second pass: infer implicit types
	for _, node := range nodes {
		tc.path = append(tc.path, node)
		if _, err := tc.inferNodeType(node); err != nil {
			return err
		}
		tc.path = tc.path[:len(tc.path)-1]
	}

	return nil
}

// collectExplicitTypes collects explicitly declared types from nodes
func (tc *TypeChecker) collectExplicitTypes(node ast.Node) error {
	switch n := node.(type) {
	case ast.FunctionNode:
		// Create new scope for function
		functionScope := &Scope{
			Parent:   tc.currentScope,
			Node:     n,
			Symbols:  make(map[ast.Identifier]Symbol),
			Children: make([]*Scope, 0),
		}

		// Register parameter types in the function scope
		for _, param := range n.Params {
			functionScope.Symbols[param.Ident.Id] = Symbol{
				Identifier: param.Ident.Id,
				Types:      []ast.TypeNode{param.Type},
				Kind:       SymbolParameter,
				Scope:      functionScope,
				Position:   tc.path,
			}
		}

		tc.currentScope = functionScope

		// Process function body
		for _, node := range n.Body {
			if err := tc.collectExplicitTypes(node); err != nil {
				return err
			}
		}

		tc.currentScope = functionScope.Parent

		// case ast.VariableDeclarationNode:
		// 	if !n.Type.IsImplicit() {
		// 		tc.currentScope.variables[n.Name] = n.Type
		// 	}

		// case ast.BlockNode:
		// 	for _, stmt := range n.Statements {
		// 		if err := tc.collectExplicitTypes(stmt); err != nil {
		// 			return err
		// 		}
		// 	}

		tc.registerFunction(n)
	}

	return nil
}

// storeInferredType associates a type with a node by storing its structural hash
func (tc *TypeChecker) storeInferredType(node ast.Node, types []ast.TypeNode) {
	hash := tc.Hasher.Hash(node)
	tc.Types[hash] = types
}

func (tc *TypeChecker) storeInferredFunctionReturnType(fn *ast.FunctionNode, returnTypes []ast.TypeNode) {
	sig := tc.Functions[fn.Id()]
	sig.ReturnTypes = returnTypes
	tc.Functions[fn.Id()] = sig
}

func (tc *TypeChecker) DebugPrintCurrentScope() {
	log.Debugf("Current scope: %s\n", tc.currentScope.Node.String())
	log.Debugf("  Defined symbols (total: %d)\n", len(tc.currentScope.Symbols))
	for _, symbol := range tc.currentScope.Symbols {
		log.Debugf("    %s: %s\n", symbol.Identifier, symbol.Types)
	}
}

func (tc *TypeChecker) GlobalScope() *Scope {
	return tc.globalScope
}
