package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"
)

// LookupInferredType looks up the inferred type of a node in the current scope
func (tc *TypeChecker) LookupInferredType(node ast.Node, requireInferred bool) ([]ast.TypeNode, error) {
	hash, err := tc.Hasher.HashNode(node)
	if err != nil {
		return nil, fmt.Errorf("failed to hash node during LookupInferredType: %s", err)
	}
	if existingType, exists := tc.Types[hash]; exists {
		if len(existingType) == 0 {
			if requireInferred {
				return nil, fmt.Errorf("expected type of node to have been inferred, found: implicit type")
			}
			return nil, nil
		}
		return existingType, nil
	}
	if requireInferred {
		return nil, fmt.Errorf("expected type of node to have been inferred, found: no registered type")
	}
	return nil, nil
}

// LookupVariableType finds a variable's type in the current scope chain
func (tc *TypeChecker) LookupVariableType(variable *ast.VariableNode, scope *Scope) (ast.TypeNode, error) {
	tc.log.Tracef("Looking up variable type for %s in scope %s", variable.Ident.ID, scope.String())

	parts := strings.Split(string(variable.Ident.ID), ".")
	baseIdent := ast.Identifier(parts[0])

	symbol, exists := scope.LookupVariable(baseIdent)
	if !exists {
		return ast.TypeNode{}, fmt.Errorf("undefined symbol: %s [scope: %s]", parts[0], scope.String())
	}

	if len(symbol.Types) != 1 {
		return ast.TypeNode{}, fmt.Errorf("expected single type for variable %s but got %d types", parts[0], len(symbol.Types))
	}

	if len(parts) == 1 {
		return symbol.Types[0], nil
	}

	currentType := symbol.Types[0]
	originalAssertion := currentType.Assertion
	for i := 1; i < len(parts); i++ {
		fieldType, err := tc.lookupFieldType(currentType, ast.Ident{ID: ast.Identifier(parts[i])}, originalAssertion)
		if err != nil {
			return ast.TypeNode{}, err
		}
		currentType = fieldType
	}

	return currentType, nil
}

// LookupEnsureBaseType looks up the base type of an ensure node in a given scope
func (tc *TypeChecker) LookupEnsureBaseType(ensure *ast.EnsureNode, scope *Scope) (*ast.TypeNode, error) {
	baseType, err := tc.LookupVariableType(&ensure.Variable, scope)
	if err != nil {
		return nil, err
	}
	return &baseType, nil
}
