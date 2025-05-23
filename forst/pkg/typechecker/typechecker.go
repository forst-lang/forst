package typechecker

import (
	"forst/pkg/ast"

	log "github.com/sirupsen/logrus"
)

type TypeChecker struct {
	// Maps structural hashes of AST nodes to their inferred or declared types
	Types map[NodeHash][]ast.TypeNode
	// Maps type identifiers to their definition nodes
	Defs map[ast.TypeIdent]ast.Node
	// Maps type identifiers to nodes where they are referenced
	Uses map[ast.TypeIdent][]ast.Node
	// Maps function identifiers to their parameter and return type signatures
	Functions     map[ast.Identifier]FunctionSignature
	Hasher        *StructuralHasher
	path          NodePath // Tracks current position while traversing AST
	scopeStack    *ScopeStack
	inferredTypes map[ast.Node][]ast.TypeNode
}

func New() *TypeChecker {
	return &TypeChecker{
		Types:         make(map[NodeHash][]ast.TypeNode),
		Defs:          make(map[ast.TypeIdent]ast.Node),
		Uses:          make(map[ast.TypeIdent][]ast.Node),
		Functions:     make(map[ast.Identifier]FunctionSignature),
		Hasher:        &StructuralHasher{},
		path:          make(NodePath, 0),
		scopeStack:    NewScopeStack(NewStructuralHasher()),
		inferredTypes: make(map[ast.Node][]ast.TypeNode),
	}
}

// Performs type inference in two passes:
// 1. Collects explicit type declarations and function signatures
// 2. Infers types for expressions and statements
func (tc *TypeChecker) CheckTypes(nodes []ast.Node) error {
	log.Trace("First pass: collecting explicit types and function signatures")
	for _, node := range nodes {
		tc.path = append(tc.path, node)
		if err := tc.collectExplicitTypes(node); err != nil {
			return err
		}
		tc.path = tc.path[:len(tc.path)-1]
	}

	for _, node := range nodes {
		tc.path = append(tc.path, node)
		if _, err := tc.inferNodeType(node); err != nil {
			return err
		}
		tc.path = tc.path[:len(tc.path)-1]
	}

	return nil
}

// Traverses the AST to gather type definitions and function signatures
func (tc *TypeChecker) collectExplicitTypes(node ast.Node) error {
	log.Tracef("Collecting explicit types for type %s", node.String())
	switch n := node.(type) {
	case ast.TypeDefNode:
		tc.registerType(n)
	case ast.FunctionNode:
		tc.pushScope(n)

		for _, param := range n.Params {
			switch p := param.(type) {
			case ast.SimpleParamNode:
				tc.storeSymbol(p.Ident.Id, []ast.TypeNode{p.Type}, SymbolVariable)
			case ast.DestructuredParamNode:
				continue
			}
		}

		for _, node := range n.Body {
			if err := tc.collectExplicitTypes(node); err != nil {
				return err
			}
		}

		tc.popScope()
		tc.registerFunction(n)
	}

	return nil
}

// Associates inferred types with an AST node using its structural hash
func (tc *TypeChecker) storeInferredType(node ast.Node, types []ast.TypeNode) {
	hash := tc.Hasher.HashNode(node)
	log.Tracef("Storing inferred type for node %s (key %s): %s", node.String(), hash.ToTypeIdent(), types)
	tc.Types[hash] = types
}

// Stores the return types for a function in its signature
func (tc *TypeChecker) storeInferredFunctionReturnType(fn *ast.FunctionNode, returnTypes []ast.TypeNode) {
	sig := tc.Functions[fn.Id()]
	sig.ReturnTypes = returnTypes
	log.Tracef("Storing inferred function return type for function %s: %s", fn.Id(), returnTypes)
	tc.Functions[fn.Id()] = sig
}

// Prints details about symbols defined in the current scope
func (tc *TypeChecker) DebugPrintCurrentScope() {
	currentScope := tc.scopeStack.CurrentScope()
	log.Debugf("Current scope: %s\n", currentScope.Node.String())
	log.Debugf("  Defined symbols (total: %d)\n", len(currentScope.Symbols))
	for _, symbol := range currentScope.Symbols {
		log.Debugf("    %s: %s\n", symbol.Identifier, symbol.Types)
	}
}

func (tc *TypeChecker) GlobalScope() *Scope {
	return tc.scopeStack.GlobalScope()
}

// Stores a type definition that will be used by code generators
// to create corresponding type definitions in the target language.
// For example, a Forst type definition like `type PhoneNumber = String.Min(3)`
// may be transformed into a TypeScript type with validation decorators.
func (tc *TypeChecker) registerType(node ast.TypeDefNode) {
	if _, exists := tc.Defs[node.Ident]; exists {
		return
	}
	tc.Defs[node.Ident] = node
}

func (tc *TypeChecker) pushScope(node ast.Node) {
	tc.scopeStack.PushScope(node)
}

func (tc *TypeChecker) popScope() {
	tc.scopeStack.PopScope()
}

// Stores a symbol definition in the current scope
func (tc *TypeChecker) storeSymbol(ident ast.Identifier, types []ast.TypeNode, kind SymbolKind) {
	currentScope := tc.scopeStack.CurrentScope()
	currentScope.Symbols[ident] = Symbol{
		Identifier: ident,
		Types:      types,
		Kind:       kind,
		Scope:      currentScope,
		Position:   tc.path,
	}
}

func (tc *TypeChecker) FindScope(node ast.Node) *Scope {
	return tc.scopeStack.FindScope(node)
}
