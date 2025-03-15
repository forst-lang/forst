package typechecker

import (
	"fmt"
	"forst/pkg/ast"
)

type TypeChecker struct {
	// Use 64-bit hashes for type mapping
	Types        map[NodeHash]ast.TypeNode
	Defs         map[*ast.Ident]ast.Node
	Uses         map[*ast.Ident][]ast.Node
	Functions    map[*ast.Ident]FunctionSignature
	hasher       *StructuralHasher
	currentScope *Scope
}

type Scope struct {
	// Parent scope for looking up variables in outer scopes
	// nil for global scope
	parent *Scope

	// Maps variable names to their identifier nodes in this scope
	// Example: "x" -> &ast.Ident{Name: "x"}
	Variables map[string]*ast.Ident

	// Maps type names to their identifier nodes in this scope
	// Example: "MyCustomType" -> &ast.Ident{Name: "MyCustomType"}
	Types map[string]*ast.Ident

	// Maps function names to their identifier nodes in this scope
	// Example: "myFunc" -> &ast.Ident{Name: "myFunc"}
	Functions map[string]*ast.Ident
}

func New() *TypeChecker {
	return &TypeChecker{
		Types:     make(map[NodeHash]ast.TypeNode),
		Defs:      make(map[*ast.Ident]ast.Node),
		Uses:      make(map[*ast.Ident][]ast.Node),
		Functions: make(map[*ast.Ident]FunctionSignature),
		currentScope: &Scope{
			Variables: make(map[string]*ast.Ident),
			Types:     make(map[string]*ast.Ident),
		},
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
		if err := tc.collectExplicitTypes(node); err != nil {
			return err
		}
	}

	// Second pass: infer implicit types and store in Types map
	for _, node := range nodes {
		// Get structural hash for the node
		hash := tc.hasher.Hash(node)

		// Check if we've already inferred this structure's type
		if _, exists := tc.Types[hash]; exists {
			continue
		}

		if inferredType, err := tc.inferTypes(node); err != nil {
			return err
		} else {
			// Store inferred type in type checker's Types map
			tc.storeType(node, *inferredType)
		}
	}

	return nil
}

// registerFunction adds a function's signature to the type checker
func (tc *TypeChecker) registerFunction(fn ast.FunctionNode) {
	ident := fn.Ident
	params := make([]ParameterSignature, len(fn.Params))
	for i, param := range fn.Params {
		params[i] = ParameterSignature{
			Ident: param.Ident,
			Type:  param.Type,
		}
	}
	tc.Functions[&ident] = FunctionSignature{
		Ident:          fn.Ident,
		Parameters:     params,
		ReturnType:     fn.ExplicitReturnType,
		IsTypeInferred: fn.ExplicitReturnType.IsImplicit(),
	}
}

// registerType adds a type declaration to the type checker
// func (tc *TypeChecker) registerType(typeDecl ast.TypeDeclarationNode) {
// 	tc.types[typeDecl.Name] = typeDecl.Type
// }

// collectExplicitTypes collects explicitly declared types from nodes
func (tc *TypeChecker) collectExplicitTypes(node ast.Node) error {
	switch n := node.(type) {
	case ast.FunctionNode:
		// Create new scope for function
		functionScope := &Scope{
			parent:    tc.currentScope,
			Variables: make(map[string]*ast.Ident),
			Types:     make(map[string]*ast.Ident),
		}

		// Register parameter types in the function scope
		for _, param := range n.Params {
			functionScope.Variables[param.Ident.Name] = &param.Ident
		}

		tc.currentScope = functionScope

		// Process function body
		if err := tc.collectExplicitTypes(n.Body); err != nil {
			return err
		}

		tc.currentScope = functionScope.parent

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
	}

	return nil
}

// pushScope creates a new scope
func (tc *TypeChecker) pushScope() {
	tc.currentScope = &Scope{
		parent:    tc.currentScope,
		Variables: make(map[string]*ast.Ident),
		Types:     make(map[string]*ast.Ident),
	}
}

// popScope returns to the parent scope
func (tc *TypeChecker) popScope() error {
	if tc.currentScope.parent == nil {
		return fmt.Errorf("cannot pop global scope")
	}
	tc.currentScope = tc.currentScope.parent
	return nil
}

// storeType associates a type with a node by storing its structural hash
func (tc *TypeChecker) storeType(node ast.Node, typ ast.TypeNode) {
	hash := tc.hasher.Hash(node)
	tc.Types[hash] = typ
}
