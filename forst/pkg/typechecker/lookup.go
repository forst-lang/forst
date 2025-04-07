package typechecker

import (
	"fmt"

	"forst/pkg/ast"
)

func findAlreadyInferredType(tc *TypeChecker, node ast.Node) ([]ast.TypeNode, error) {
	hash := tc.Hasher.Hash(node)
	if existingType, exists := tc.Types[hash]; exists {
		// Ignore types that are still marked as implicit, as they are not yet inferred
		if len(existingType) > 0 {
			return nil, nil
		}
		return existingType, nil
	}
	return nil, nil
}

// lookupVariableType finds a variable's type in the current scope chain
func (tc *TypeChecker) LookupVariableType(variable *ast.VariableNode, currentScope *Scope) (ast.TypeNode, error) {
	symbol, err := tc.lookupSymbol(variable.Ident.Id, currentScope)
	if err != nil {
		return ast.TypeNode{}, err
	}
	if len(symbol.Types) != 1 {
		return ast.TypeNode{}, fmt.Errorf("expected single type for variable %s but got %d types", variable.Ident.String(), len(symbol.Types))
	}
	return symbol.Types[0], nil
}

func (tc *TypeChecker) lookupSymbol(ident ast.Identifier, currentScope *Scope) (*Symbol, error) {
	for scope := currentScope; scope != nil; scope = scope.Parent {
		if sym, exists := scope.Symbols[ident]; exists {
			return &sym, nil
		}
	}
	return nil, fmt.Errorf("undefined symbol: %s", ident)
}

func (tc *TypeChecker) LookupFunctionReturnType(function *ast.FunctionNode, currentScope *Scope) ([]ast.TypeNode, error) {
	sig, exists := tc.Functions[function.Id()]
	if !exists {
		return nil, fmt.Errorf("undefined function: %s", function.Ident)
	}
	return sig.ReturnTypes, nil
}

func (tc *TypeChecker) LookupEnsureBaseType(ensure *ast.EnsureNode, currentScope *Scope) (*ast.TypeNode, error) {
	baseType, err := tc.LookupVariableType(&ensure.Variable, currentScope)
	if err != nil {
		return nil, err
	}
	return &baseType, nil
}

func (tc *TypeChecker) LookupAssertionType(assertion *ast.AssertionNode, currentScope *Scope) (*ast.TypeNode, error) {
	hash := tc.Hasher.Hash(assertion)
	if existingType, exists := tc.Types[hash]; exists {
		if len(existingType) != 1 {
			return nil, fmt.Errorf("expected single type for assertion %s but got %d types", typeIdent(hash), len(existingType))
		}
		return &existingType[0], nil
	}
	return &ast.TypeNode{
		Name:      typeIdent(hash),
		Assertion: assertion,
	}, nil
}
