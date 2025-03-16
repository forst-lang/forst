package typechecker

import (
	"fmt"
	"forst/pkg/ast"
)

type TypeChecker struct {
	// Use value-based Ident keys instead of pointers
	Types        map[NodeHash]ast.TypeNode
	Defs         map[ast.Identifier]ast.Node
	Uses         map[ast.Identifier][]ast.Node
	Functions    map[ast.Identifier]FunctionSignature
	hasher       *StructuralHasher
	currentScope *Scope
	Scopes       map[ast.Node]*Scope // Map AST nodes to their scopes
	path         NodePath            // Track current position in AST
}

func New() *TypeChecker {
	globalScope := &Scope{
		Parent:  nil,
		Symbols: make(map[ast.Identifier]Symbol),
	}

	return &TypeChecker{
		Types:        make(map[NodeHash]ast.TypeNode),
		Defs:         make(map[ast.Identifier]ast.Node),
		Uses:         make(map[ast.Identifier][]ast.Node),
		Functions:    make(map[ast.Identifier]FunctionSignature),
		hasher:       &StructuralHasher{},
		currentScope: globalScope,
		Scopes:       make(map[ast.Node]*Scope),
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
				Type:       param.Type,
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
func (tc *TypeChecker) storeInferredType(node ast.Node, typ ast.TypeNode) {
	hash := tc.hasher.Hash(node)
	tc.Types[hash] = typ
}

func (tc *TypeChecker) storeInferredFunctionReturnType(fn *ast.FunctionNode, typ ast.TypeNode) {
	sig := tc.Functions[fn.Id()]
	sig.ReturnType = typ
	tc.Functions[fn.Id()] = sig
}

func (tc *TypeChecker) DebugPrintCurrentScope() {
	fmt.Printf("Current scope: %s\n", tc.currentScope.Node.String())
	fmt.Printf("  Defined symbols (total: %d)\n", len(tc.currentScope.Symbols))
	for _, symbol := range tc.currentScope.Symbols {
		fmt.Printf("    %s: %s\n", symbol.Identifier, symbol.Type)
	}
}
