// Package typechecker performs type inference and type checking on the AST
package typechecker

import (
	"fmt"
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
	Functions  map[ast.Identifier]FunctionSignature
	Hasher     *hasher.StructuralHasher
	path       NodePath // Tracks current position while traversing AST
	scopeStack *ScopeStack
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
	// Whether to report phases
	reportPhases bool
}

// New creates a new TypeChecker
func New(log *logrus.Logger, reportPhases bool) *TypeChecker {
	if log == nil {
		log = logrus.New()
		log.Warnf("No logger provided, using default logger")
	}
	h := hasher.New()
	tc := &TypeChecker{
		Types:               make(map[NodeHash][]ast.TypeNode),
		Defs:                make(map[ast.TypeIdent]ast.Node),
		Uses:                make(map[ast.TypeIdent][]ast.Node),
		Functions:           make(map[ast.Identifier]FunctionSignature),
		Hasher:              h,
		path:                make(NodePath, 0),
		scopeStack:          NewScopeStack(h, log),
		InferredTypes:       make(map[NodeHash][]ast.TypeNode),
		VariableTypes:       make(map[ast.Identifier][]ast.TypeNode),
		FunctionReturnTypes: make(map[ast.Identifier][]ast.TypeNode),
		log:                 log,
		reportPhases:        reportPhases,
	}

	return tc
}

// CheckTypes performs type inference in two passes:
// 1. Collects explicit type declarations and function signatures
// 2. Infers types for expressions and statements
func (tc *TypeChecker) CheckTypes(nodes []ast.Node) error {
	if tc.reportPhases {
		tc.log.WithFields(logrus.Fields{
			"function": "CheckTypes",
		}).Info("First pass: collecting explicit types and function signatures")
	}

	for _, node := range nodes {
		tc.path = append(tc.path, node)
		if err := tc.collectExplicitTypes(node); err != nil {
			return err
		}
		tc.path = tc.path[:len(tc.path)-1]
	}

	tc.log.WithFields(logrus.Fields{
		"imports":   len(tc.imports),
		"typeDefs":  len(tc.Defs),
		"functions": len(tc.Functions),
		"uses":      len(tc.Uses),
		"function":  "CheckTypes",
	}).Debug("Collected types and function signatures")

	if tc.reportPhases {
		tc.log.WithFields(logrus.Fields{
			"function": "CheckTypes",
		}).Info("Starting second pass: inferring types")
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

// Associates inferred types with an AST node using its structural hash
func (tc *TypeChecker) storeInferredType(node ast.Node, types []ast.TypeNode) {
	hash, err := tc.Hasher.HashNode(node)
	if err != nil {
		tc.log.WithFields(logrus.Fields{
			"node":     node.String(),
			"function": "storeInferredType",
		}).WithError(err).Error("failed to hash node during storeInferredType")
		return
	}
	tc.Types[hash] = types
	tc.log.WithFields(logrus.Fields{
		"node":     node.String(),
		"key":      hash.ToTypeIdent(),
		"types":    types,
		"function": "storeInferredType",
		"hash":     fmt.Sprintf("%x", uint64(hash)),
	}).Trace("Stored inferred type for node")
}

// resolveAliasedType resolves a type to its aliased name if it's a hash-based type that matches a user-defined type
func (tc *TypeChecker) resolveAliasedType(typeNode ast.TypeNode) ast.TypeNode {
	// If this is a hash-based type, check for structural identity with user-defined types
	if typeNode.TypeKind == ast.TypeKindHashBased {
		// Look up the shape for this hash-based type
		hashDef, hashExists := tc.Defs[typeNode.Ident]
		if hashExists {
			if hashTypeDef, ok := hashDef.(ast.TypeDefNode); ok {
				if hashShapeExpr, ok := hashTypeDef.Expr.(ast.TypeDefShapeExpr); ok {
					hashShape := hashShapeExpr.Shape
					for _, def := range tc.Defs {
						if userDef, ok := def.(ast.TypeDefNode); ok && userDef.Ident != "" {
							if shapeExpr, ok := userDef.Expr.(ast.TypeDefShapeExpr); ok {
								userShape := shapeExpr.Shape
								if tc.shapesAreStructurallyIdentical(hashShape, userShape) {
									tc.log.WithFields(logrus.Fields{
										"hashType":    typeNode.Ident,
										"aliasedType": userDef.Ident,
										"function":    "resolveAliasedType",
									}).Debug("Resolved hash-based type to aliased type")
									return ast.TypeNode{
										Ident:      userDef.Ident,
										TypeKind:   typeNode.TypeKind,
										Assertion:  typeNode.Assertion,
										TypeParams: typeNode.TypeParams,
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return typeNode
}

// GetAliasedTypeName returns the aliased type name for any type node, ensuring consistent type aliasing
// This is the unified function that should be used by both typechecker and transformer
func (tc *TypeChecker) GetAliasedTypeName(typeNode ast.TypeNode) (string, error) {
	// Handle built-in types
	if tc.IsGoBuiltinType(string(typeNode.Ident)) || typeNode.Ident == ast.TypeString || typeNode.Ident == ast.TypeInt || typeNode.Ident == ast.TypeFloat || typeNode.Ident == ast.TypeBool || typeNode.Ident == ast.TypeVoid || typeNode.Ident == ast.TypeError {
		// Convert Forst built-in types to Go built-in types
		switch typeNode.Ident {
		case ast.TypeString:
			return "string", nil
		case ast.TypeInt:
			return "int", nil
		case ast.TypeFloat:
			return "float64", nil
		case ast.TypeBool:
			return "bool", nil
		case ast.TypeVoid:
			return "", nil
		case ast.TypeError:
			return "error", nil
		default:
			return string(typeNode.Ident), nil
		}
	}

	// Handle pointer types
	if typeNode.Ident == ast.TypePointer {
		if len(typeNode.TypeParams) == 0 {
			return "", fmt.Errorf("pointer type must have a base type parameter")
		}
		baseTypeName, err := tc.GetAliasedTypeName(typeNode.TypeParams[0])
		if err != nil {
			return "", fmt.Errorf("failed to get base type name for pointer: %w", err)
		}
		return "*" + baseTypeName, nil
	}

	// If this is a hash-based type, check for structural identity with user-defined types
	if typeNode.TypeKind == ast.TypeKindHashBased {
		// Look up the shape for this hash-based type
		hashDef, hashExists := tc.Defs[typeNode.Ident]
		if hashExists {
			if hashTypeDef, ok := hashDef.(ast.TypeDefNode); ok {
				if hashShapeExpr, ok := hashTypeDef.Expr.(ast.TypeDefShapeExpr); ok {
					hashShape := hashShapeExpr.Shape
					for _, def := range tc.Defs {
						if userDef, ok := def.(ast.TypeDefNode); ok && userDef.Ident != "" {
							if shapeExpr, ok := userDef.Expr.(ast.TypeDefShapeExpr); ok {
								userShape := shapeExpr.Shape
								if tc.shapesAreStructurallyIdentical(hashShape, userShape) {
									tc.log.WithFields(logrus.Fields{
										"hashType":    typeNode.Ident,
										"aliasedType": userDef.Ident,
										"function":    "GetAliasedTypeName",
									}).Debug("Resolved hash-based type to aliased type")
									return string(userDef.Ident), nil
								}
							}
						}
					}
				}
			}
		}
	}

	// For user-defined types, check if they're already defined in the typechecker
	if typeNode.Ident != "" {
		if _, exists := tc.Defs[typeNode.Ident]; exists {
			return string(typeNode.Ident), nil
		}
	}

	// If not found in Defs, fall back to hash-based name
	hash, err := tc.Hasher.HashNode(typeNode)
	if err != nil {
		return "", fmt.Errorf("failed to hash type node: %w", err)
	}
	return string(hash.ToTypeIdent()), nil
}

// IsGoBuiltinType checks if a type is a Go builtin type
func (tc *TypeChecker) IsGoBuiltinType(typeName string) bool {
	builtinTypes := map[string]bool{
		"string":  true,
		"int":     true,
		"int8":    true,
		"int16":   true,
		"int32":   true,
		"int64":   true,
		"uint":    true,
		"uint8":   true,
		"uint16":  true,
		"uint32":  true,
		"uint64":  true,
		"float32": true,
		"float64": true,
		"bool":    true,
		"byte":    true,
		"rune":    true,
		"error":   true,
	}
	return builtinTypes[typeName]
}

// Stores the return types for a function in its signature
func (tc *TypeChecker) storeInferredFunctionReturnType(fn *ast.FunctionNode, returnTypes []ast.TypeNode) {
	// Resolve aliased types for return types
	resolvedReturnTypes := make([]ast.TypeNode, len(returnTypes))
	for i, returnType := range returnTypes {
		resolvedReturnTypes[i] = tc.resolveAliasedType(returnType)
	}

	sig := tc.Functions[fn.Ident.ID]
	sig.ReturnTypes = resolvedReturnTypes
	tc.Functions[fn.Ident.ID] = sig
	tc.log.WithFields(logrus.Fields{
		"fn":          fn.Ident.ID,
		"returnTypes": resolvedReturnTypes,
		"sig":         sig,
		"function":    "storeInferredFunctionReturnType",
	}).Trace("Stored inferred function return type")
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
		tc.log.WithFields(logrus.Fields{
			"scope": currentScope.String(),
			"addr":  fmt.Sprintf("%p", currentScope),
		}).Debug("Current scope")
	}

	tc.log.WithFields(logrus.Fields{
		"total": len(currentScope.Symbols),
	}).Debug("Defined symbols")

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

	tc.log.WithFields(logrus.Fields{
		"scope":    scope.String(),
		"addr":     fmt.Sprintf("%p", scope),
		"function": "pushScope",
	}).Debug("Pushed scope")
	return scope
}

// popScope removes the current scope and returns to the parent scope
// Intended for use in the collection pass of the typechecker, not the transformer
func (tc *TypeChecker) popScope() {
	currentScope := tc.CurrentScope()
	tc.scopeStack.popScope()

	tc.log.WithFields(logrus.Fields{
		"scope":    currentScope.String(),
		"addr":     fmt.Sprintf("%p", currentScope),
		"function": "popScope",
	}).Debug("Popped scope")
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

	tc.log.WithFields(logrus.Fields{
		"ident":    ident,
		"types":    types,
		"scope":    currentScope.String(),
		"addr":     fmt.Sprintf("%p", currentScope),
		"function": "storeSymbol",
	}).Trace("Stored symbol")
}

// CurrentScope returns the current scope
func (tc *TypeChecker) CurrentScope() *Scope {
	return tc.scopeStack.currentScope()
}

// RegisterTypeIfMissing registers a type definition if not already present in Defs.
// Accepts either ast.TypeDefNode or ast.TypeDefShapeExpr as def.
func (tc *TypeChecker) RegisterTypeIfMissing(ident ast.TypeIdent, def interface{}) {
	if _, exists := tc.Defs[ident]; exists {
		return
	}
	switch d := def.(type) {
	case ast.TypeDefNode:
		tc.Defs[ident] = d
	case ast.TypeDefShapeExpr:
		tc.Defs[ident] = d
	default:
		panic("RegisterTypeIfMissing: unsupported type definition")
	}
}
