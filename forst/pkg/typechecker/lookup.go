package typechecker

import (
	"fmt"

	"forst/pkg/ast"
)

// lookupVariableType finds a variable's type in the current scope chain
func (tc *TypeChecker) LookupVariableType(variable *ast.VariableNode) (ast.TypeNode, error) {
	symbol, err := tc.lookupSymbol(variable.Ident.Id)
	if err != nil {
		return ast.TypeNode{}, err
	}
	if len(symbol.Types) != 1 {
		return ast.TypeNode{}, fmt.Errorf("expected single type for variable %s but got %d types", variable.Ident.String(), len(symbol.Types))
	}
	return symbol.Types[0], nil
}

func (tc *TypeChecker) lookupSymbol(ident ast.Identifier) (*Symbol, error) {
	for scope := tc.currentScope; scope != nil; scope = scope.Parent {
		if sym, exists := scope.Symbols[ident]; exists {
			return &sym, nil
		}
	}
	return nil, fmt.Errorf("undefined symbol: %s", ident)
}

func (tc *TypeChecker) LookupFunctionReturnType(function *ast.FunctionNode) ([]ast.TypeNode, error) {
	symbol, err := tc.lookupSymbol(function.Ident.Id)
	if err != nil {
		return nil, err
	}
	if symbol.Kind != SymbolFunction {
		return nil, fmt.Errorf("%s is not a function", function.Ident)
	}
	return symbol.Types, nil
}

func (tc *TypeChecker) LookupAssertionType(ensure *ast.EnsureNode) (*ast.TypeNode, error) {
	baseType, err := tc.LookupVariableType(&ensure.Variable)
	if err != nil {
		return nil, err
	}
	return &baseType, nil
}
