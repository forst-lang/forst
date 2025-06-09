package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// LookupFunctionReturnType looks up the return type of a function node
func (tc *TypeChecker) LookupFunctionReturnType(function *ast.FunctionNode) ([]ast.TypeNode, error) {
	sig, exists := tc.Functions[function.Ident.ID]
	if !exists {
		return nil, fmt.Errorf("undefined function: %s", function.Ident)
	}
	return sig.ReturnTypes, nil
}

// LookupAssertionType looks up the type of an assertion node
func (tc *TypeChecker) LookupAssertionType(assertion *ast.AssertionNode) (*ast.TypeNode, error) {
	hash, err := tc.Hasher.HashNode(assertion)
	if err != nil {
		return nil, fmt.Errorf("failed to hash assertion during LookupAssertionType: %s", err)
	}
	if existingType, exists := tc.Types[hash]; exists {
		if len(existingType) != 1 {
			return nil, fmt.Errorf("expected single type for assertion %s but got %d types", hash.ToTypeIdent(), len(existingType))
		}
		return &existingType[0], nil
	}

	typeNode := &ast.TypeNode{
		Ident:     hash.ToTypeIdent(),
		Assertion: assertion,
	}
	tc.storeInferredType(assertion, []ast.TypeNode{*typeNode})
	return typeNode, nil
}
