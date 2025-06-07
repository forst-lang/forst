// Package typechecker performs type inference and type checking on the AST
package typechecker

import (
	"forst/internal/ast"

	log "github.com/sirupsen/logrus"
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
	Hasher        *StructuralHasher
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
}

// New creates a new TypeChecker
func New() *TypeChecker {
	return &TypeChecker{
		Types:               make(map[NodeHash][]ast.TypeNode),
		Defs:                make(map[ast.TypeIdent]ast.Node),
		Uses:                make(map[ast.TypeIdent][]ast.Node),
		Functions:           make(map[ast.Identifier]FunctionSignature),
		Hasher:              &StructuralHasher{},
		path:                make(NodePath, 0),
		scopeStack:          NewScopeStack(NewStructuralHasher()),
		inferredTypes:       make(map[ast.Node][]ast.TypeNode),
		InferredTypes:       make(map[NodeHash][]ast.TypeNode),
		VariableTypes:       make(map[ast.Identifier][]ast.TypeNode),
		FunctionReturnTypes: make(map[ast.Identifier][]ast.TypeNode),
	}
}

// CheckTypes performs type inference in two passes:
// 1. Collects explicit type declarations and function signatures
// 2. Infers types for expressions and statements
func (tc *TypeChecker) CheckTypes(nodes []ast.Node) error {
	log.Trace("[CheckTypes] First pass: collecting explicit types and function signatures")
	for _, node := range nodes {
		tc.path = append(tc.path, node)
		if err := tc.collectExplicitTypes(node); err != nil {
			return err
		}
		tc.path = tc.path[:len(tc.path)-1]
	}

	log.Debugf("Collected imports: %v", tc.imports)

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
	log.Tracef("[collectExplicitTypes] Collecting explicit types for type %s", node.String())
	switch n := node.(type) {
	case ast.ImportNode:
		log.Debugf("[collectExplicitTypes] Collecting import: %v", n)
		tc.imports = append(tc.imports, n)
	case ast.ImportGroupNode:
		log.Debugf("[collectExplicitTypes] Collecting import group: %v", n)
		tc.imports = append(tc.imports, n.Imports...)
	case ast.TypeDefNode:
		tc.registerType(n)
	case ast.FunctionNode:
		tc.PushScope(n)

		for _, param := range n.Params {
			switch p := param.(type) {
			case ast.SimpleParamNode:
				tc.storeSymbol(p.Ident.ID, []ast.TypeNode{p.Type}, SymbolVariable)
			case ast.DestructuredParamNode:
				// TODO: Handle destructured params
				continue
			}
		}

		for _, node := range n.Body {
			if err := tc.collectExplicitTypes(node); err != nil {
				return err
			}
		}

		tc.PopScope()
		tc.registerFunction(n)
	case *ast.TypeGuardNode:
		tc.PushScope(n)

		tc.PopScope()
		tc.registerTypeGuard(*n)
	}

	return nil
}

// Associates inferred types with an AST node using its structural hash
func (tc *TypeChecker) storeInferredType(node ast.Node, types []ast.TypeNode) {
	hash := tc.Hasher.HashNode(node)
	log.Tracef("[storeInferredType] Storing inferred type for node %s (key %s): %s", node.String(), hash.ToTypeIdent(), types)
	tc.Types[hash] = types
}

// Stores the return types for a function in its signature
func (tc *TypeChecker) storeInferredFunctionReturnType(fn *ast.FunctionNode, returnTypes []ast.TypeNode) {
	sig := tc.Functions[fn.Ident.ID]
	sig.ReturnTypes = returnTypes
	log.Tracef("[storeInferredFunctionReturnType] Storing inferred function return type for function %s: %s", fn.Ident.ID, returnTypes)
	tc.Functions[fn.Ident.ID] = sig
}

// DebugPrintCurrentScope prints details about symbols defined in the current scope
func (tc *TypeChecker) DebugPrintCurrentScope() {
	currentScope := tc.scopeStack.CurrentScope()
	if currentScope == nil {
		log.Debug("Current scope is nil")
		return
	}
	if currentScope.Node == nil {
		log.Debug("Current scope node is nil")
	} else {
		log.Debugf("Current scope: %s\n", (*currentScope.Node).String())
	}
	log.Debugf("  Defined symbols (total: %d)\n", len(currentScope.Symbols))
	for _, symbol := range currentScope.Symbols {
		log.Debugf("    %s: %s\n", symbol.Identifier, symbol.Types)
	}
}

// GlobalScope returns the root scope
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
	// Store the type definition node
	tc.Defs[node.Ident] = node
	log.Tracef("[registerType] Registered type %s: %+v", node.Ident, node)

	// If this is a shape type, also store the underlying ShapeNode for field access
	if assertionExpr, ok := node.Expr.(ast.TypeDefAssertionExpr); ok {
		if assertionExpr.Assertion != nil {
			// If this is a direct shape alias (e.g. type AppContext = { ... })
			if assertionExpr.Assertion.BaseType != nil && *assertionExpr.Assertion.BaseType == ast.TypeShape {
				// If there are no constraints, check if the assertion has a Match constraint with a shape
				if len(assertionExpr.Assertion.Constraints) == 0 && assertionExpr.Assertion.BaseType != nil {
					// See if the assertion itself has a shape (Match constraint)
					if assertionExpr.Assertion != nil && assertionExpr.Assertion.Constraints != nil {
						for _, constraint := range assertionExpr.Assertion.Constraints {
							for _, arg := range constraint.Args {
								if arg.Shape != nil {
									tc.registerShapeType(node.Ident, *arg.Shape)
								}
							}
						}
					}
				}
				// Try to extract from constraints if present (existing logic)
				for _, constraint := range assertionExpr.Assertion.Constraints {
					for _, arg := range constraint.Args {
						if arg.Shape != nil {
							tc.registerShapeType(node.Ident, *arg.Shape)
						}
					}
				}
			}
		}
	} else if shapeExpr, ok := node.Expr.(ast.TypeDefShapeExpr); ok {
		// If the type definition is directly a shape, store it with a special key
		tc.registerShapeType(node.Ident, shapeExpr.Shape)
	}
}

// registerShapeType registers a shape type with its fields
func (tc *TypeChecker) registerShapeType(ident ast.TypeIdent, shape ast.ShapeNode) {
	tc.Defs[ident] = ast.TypeDefNode{
		Ident: ident,
		Expr: ast.TypeDefShapeExpr{
			Shape: shape,
		},
	}
	log.Tracef("[registerShapeType] Registered shape type %s: %+v", ident, shape)
}

// PushScope creates a new scope for the given node
func (tc *TypeChecker) PushScope(node ast.Node) *Scope {
	return tc.scopeStack.PushScope(node)
}

// PopScope removes the current scope and returns to the parent scope
func (tc *TypeChecker) PopScope() {
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

// FindScope finds the scope for a given node
func (tc *TypeChecker) FindScope(node ast.Node) *Scope {
	return tc.scopeStack.FindScope(node)
}

// CurrentScope returns the current scope
func (tc *TypeChecker) CurrentScope() *Scope {
	return tc.scopeStack.CurrentScope()
}
