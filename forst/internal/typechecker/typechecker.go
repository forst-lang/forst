// Package typechecker performs type inference and type checking on the AST
package typechecker

import (
	"forst/internal/ast"
	"forst/internal/hasher"

	"github.com/sirupsen/logrus"
)

// TypeChecker performs type inference and type checking on the AST
type TypeChecker struct {
	// Maps structural hashes of AST nodes to their inferred or declared types
	Types map[NodeHash][]ast.TypeNode
	// Maps type identifiers to their definition nodes
	Defs map[ast.TypeIdent]ast.Node
	// Maps type identifiers to nodes where they are referenced
	Uses map[ast.TypeIdent][]ast.Node
	// Maps function identifiers to their parameter and return type signatures
	Functions     map[ast.Identifier]FunctionSignature
	Hasher        *hasher.StructuralHasher
	path          NodePath // Tracks current position while traversing AST
	scopeStack    *ScopeStack
	inferredTypes map[ast.Node][]ast.TypeNode
	// Map of inferred types for nodes
	InferredTypes map[NodeHash][]ast.TypeNode
	// Map of inferred variable types
	VariableTypes map[ast.Identifier][]ast.TypeNode
	// Map of inferred function return types
	FunctionReturnTypes map[ast.Identifier][]ast.TypeNode
	// List of imported packages
	imports []ast.ImportNode
	// Logger for the type checker
	log *logrus.Logger
}

// New creates a new TypeChecker
func New(log *logrus.Logger) *TypeChecker {
	if log == nil {
		log = logrus.New()
		log.Warnf("No logger provided, using default logger")
	}
	h := hasher.New()
	return &TypeChecker{
		Types:               make(map[NodeHash][]ast.TypeNode),
		Defs:                make(map[ast.TypeIdent]ast.Node),
		Uses:                make(map[ast.TypeIdent][]ast.Node),
		Functions:           make(map[ast.Identifier]FunctionSignature),
		Hasher:              h,
		path:                make(NodePath, 0),
		scopeStack:          NewScopeStack(h, log),
		inferredTypes:       make(map[ast.Node][]ast.TypeNode),
		InferredTypes:       make(map[NodeHash][]ast.TypeNode),
		VariableTypes:       make(map[ast.Identifier][]ast.TypeNode),
		FunctionReturnTypes: make(map[ast.Identifier][]ast.TypeNode),
		log:                 log,
	}
}

// CheckTypes performs type inference in two passes:
// 1. Collects explicit type declarations and function signatures
// 2. Infers types for expressions and statements
func (tc *TypeChecker) CheckTypes(nodes []ast.Node) error {
	tc.log.Info("[CheckTypes] First pass: collecting explicit types and function signatures")
	for _, node := range nodes {
		tc.path = append(tc.path, node)
		if err := tc.collectExplicitTypes(node); err != nil {
			return err
		}
		tc.path = tc.path[:len(tc.path)-1]
	}
	tc.log.Infof("[CheckTypes] Collected %d imports, %d type defs, %d functions, %d type guards",
		len(tc.imports), len(tc.Defs), len(tc.Functions), len(tc.Uses))
	tc.log.Infof("[CheckTypes] Starting second pass: inferring types")

	for _, node := range nodes {
		tc.path = append(tc.path, node)
		if _, err := tc.inferNodeType(node); err != nil {
			return err
		}
		tc.path = tc.path[:len(tc.path)-1]
	}

	return nil
}

// Associates inferred types with an AST node using its structural hash
func (tc *TypeChecker) storeInferredType(node ast.Node, types []ast.TypeNode) {
	hash, err := tc.Hasher.HashNode(node)
	if err != nil {
		tc.log.WithError(err).Error("failed to hash node during storeInferredType")
		return
	}
	tc.Types[hash] = types
	tc.log.Tracef("[storeInferredType] Stored inferred type for node %s (key %s): %s", node.String(), hash.ToTypeIdent(), types)
}

// Stores the return types for a function in its signature
func (tc *TypeChecker) storeInferredFunctionReturnType(fn *ast.FunctionNode, returnTypes []ast.TypeNode) {
	sig := tc.Functions[fn.Ident.ID]
	sig.ReturnTypes = returnTypes
	tc.Functions[fn.Ident.ID] = sig
	tc.log.Tracef("[storeInferredFunctionReturnType] Stored inferred function return type for function %s: %s", fn.Ident.ID, returnTypes)
}

// DebugPrintCurrentScope prints details about symbols defined in the current scope
func (tc *TypeChecker) DebugPrintCurrentScope() {
	currentScope := tc.scopeStack.currentScope()
	if currentScope == nil {
		tc.log.Debug("Current scope is nil")
		return
	}
	if currentScope.Node == nil {
		tc.log.Debug("Current scope node is nil")
	} else {
		tc.log.Debugf("Current scope: %s (%p)\n", currentScope.String(), currentScope)
	}
	tc.log.Debugf("  Defined symbols (total: %d)\n", len(currentScope.Symbols))
	for _, symbol := range currentScope.Symbols {
		tc.log.Debugf("    %s: %s\n", symbol.Identifier, symbol.Types)
	}
}

// globalScope returns the root scope
func (tc *TypeChecker) globalScope() *Scope {
	return tc.scopeStack.globalScope()
}

// pushScope creates a new scope for the given node
// Intended for use in the collection pass of the typechecker, not the transformer
func (tc *TypeChecker) pushScope(node ast.Node) *Scope {
	scope := tc.scopeStack.pushScope(node)
	tc.log.Debugf("[pushScope] Pushed scope %s (%p)", scope.String(), scope)
	return scope
}

// popScope removes the current scope and returns to the parent scope
// Intended for use in the collection pass of the typechecker, not the transformer
func (tc *TypeChecker) popScope() {
	currentScope := tc.CurrentScope()
	tc.scopeStack.popScope()
	tc.log.Debugf("[popScope] Popped scope %s (%p)", currentScope.String(), currentScope)
}

// RestoreScope restores the scope for a given node
// Intended for use after the collection pass of the typechecker has completed
func (tc *TypeChecker) RestoreScope(node ast.Node) error {
	return tc.scopeStack.restoreScope(node)
}

// Stores a symbol definition in the current scope
func (tc *TypeChecker) storeSymbol(ident ast.Identifier, types []ast.TypeNode, kind SymbolKind) {
	currentScope := tc.CurrentScope()
	currentScope.Symbols[ident] = Symbol{
		Identifier: ident,
		Types:      types,
		Kind:       kind,
		Scope:      currentScope,
		Position:   tc.path,
	}
	tc.log.Tracef("[storeSymbol] Stored symbol '%s' with types %v in scope %s (%p)", ident, types, currentScope.String(), currentScope)
}

// CurrentScope returns the current scope
func (tc *TypeChecker) CurrentScope() *Scope {
	return tc.scopeStack.currentScope()
}
