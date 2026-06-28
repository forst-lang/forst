package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// LookupFunctionReturnType looks up the return type of a function node
func (tc *TypeChecker) LookupFunctionReturnType(function *ast.FunctionNode) ([]ast.TypeNode, error) {
	if function.Receiver != nil {
		recvType := receiverTypeIdentFromFn(function)
		if sig, ok := tc.lookupTypeMethod(recvType, string(function.Ident.ID)); ok {
			return sig.ReturnTypes, nil
		}
		if len(function.ReturnTypes) > 0 {
			return function.ReturnTypes, nil
		}
		return nil, fmt.Errorf("undefined receiver method: %s on %s", function.Ident.ID, recvType)
	}
	sig, exists := tc.Functions[function.Ident.ID]
	if !exists {
		return nil, fmt.Errorf("undefined function: %s", function.Ident.ID)
	}
	return sig.ReturnTypes, nil
}

// LookupAssertionType looks up the type of an assertion node
func (tc *TypeChecker) LookupAssertionType(assertion *ast.AssertionNode) (*ast.TypeNode, error) {
	// If the assertion is just a base type (e.g., a type guard), return that type directly
	if assertion != nil && assertion.BaseType != nil && len(assertion.Constraints) == 0 {
		baseType := *assertion.BaseType

		// For user-defined types, return the original name
		// For hash-based types, return the hash-based name
		typeNode := ast.TypeNode{Ident: baseType}
		if typeNode.IsHashBased() {
			// For hash-based types, get the hash-based name
			if def, exists := tc.Defs[baseType]; exists {
				hash, err := tc.Hasher.HashNode(def)
				if err != nil {
					return nil, fmt.Errorf("failed to hash type definition during LookupAssertionType: %s", err)
				}
				return &ast.TypeNode{Ident: hash.ToTypeIdent()}, nil
			}
		}

		// For user-defined types and built-in types, return the original name
		return &ast.TypeNode{Ident: baseType}, nil
	}

	hash, err := tc.Hasher.HashNode(assertion)
	if err != nil {
		return nil, fmt.Errorf("failed to hash assertion during LookupAssertionType: %s", err)
	}
	if existingType, exists := tc.Types[hash]; exists {
		if len(existingType) != 1 {
			return nil, fmt.Errorf("expected single type for assertion %s but got %d types", hash.ToTypeIdent(), len(existingType))
		}
		// Always return the hash-based name for structural types
		return &ast.TypeNode{Ident: hash.ToTypeIdent()}, nil
	}

	typeNode := &ast.TypeNode{
		Ident:     hash.ToTypeIdent(),
		Assertion: assertion,
		TypeKind:  ast.TypeKindHashBased, // Set the correct type kind for hash-based types
	}
	tc.storeInferredType(assertion, []ast.TypeNode{*typeNode})
	return typeNode, nil
}
