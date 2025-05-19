package typechecker

import (
	"forst/pkg/ast"

	log "github.com/sirupsen/logrus"
)

type TypeChecker struct {
	// Map structural hashes of nodes to their types
	Types map[NodeHash][]ast.TypeNode
	// Map type identifiers to their definitions
	Defs map[ast.TypeIdent]ast.Node
	// Map type identifiers to their uses
	Uses map[ast.TypeIdent][]ast.Node
	// Map function identifiers to their signatures
	Functions map[ast.Identifier]FunctionSignature
	// Hasher for structural hashing of AST nodes
	Hasher *StructuralHasher
	// Current scope
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
		Defs:         make(map[ast.TypeIdent]ast.Node),
		Uses:         make(map[ast.TypeIdent][]ast.Node),
		Functions:    make(map[ast.Identifier]FunctionSignature),
		Hasher:       &StructuralHasher{},
		currentScope: globalScope,
		globalScope:  globalScope,
		scopes:       make(map[NodeHash]*Scope),
		path:         make(NodePath, 0),
	}
}

// CheckTypes performs type inference and collects type information
func (tc *TypeChecker) CheckTypes(nodes []ast.Node) error {
	// First pass: collect function signatures and explicit types
	log.Trace("First pass: collecting explicit types and function signatures")
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
	log.Tracef("Collecting explicit types for type %s", node.String())
	switch n := node.(type) {
	case ast.TypeDefNode:
		tc.registerType(n)
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
			switch p := param.(type) {
			case ast.SimpleParamNode:
				functionScope.Symbols[p.Ident.Id] = Symbol{
					Identifier: p.Ident.Id,
					Types:      []ast.TypeNode{p.Type},
					Kind:       SymbolParameter,
					Scope:      functionScope,
					Position:   tc.path,
				}
			case ast.DestructuredParamNode:
				// Handle destructured params if needed
				continue
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
	hash := tc.Hasher.HashNode(node)
	log.Tracef("Storing inferred type for node %s (key %s): %s", node.String(), hash.ToTypeIdent(), types)
	tc.Types[hash] = types
}

func (tc *TypeChecker) storeInferredFunctionReturnType(fn *ast.FunctionNode, returnTypes []ast.TypeNode) {
	sig := tc.Functions[fn.Id()]
	sig.ReturnTypes = returnTypes
	log.Tracef("Storing inferred function return type for function %s: %s", fn.Id(), returnTypes)
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

// registerType stores a type definition in the TypeChecker's Defs map.
//
// The type definition will be used by code generators to create corresponding
// type definitions in the target language. For example, a Forst type definition
// like `type PhoneNumber = String.Min(3)` may be transformed into a TypeScript
// type with validation decorators.
//
// Parameters:
//   - node: The TypeDefNode containing the type definition to register
//
// The function silently returns if the type is already registered.
func (tc *TypeChecker) registerType(node ast.TypeDefNode) {
	if _, exists := tc.Defs[node.Ident]; exists {
		// panic(fmt.Sprintf("type %s already defined", node.Ident))
		return
	}
	tc.Defs[node.Ident] = node
}
