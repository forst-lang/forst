package typechecker

import (
	"fmt"

	"forst/pkg/ast"
)

// Enhanced variable lookup that can find where a variable was defined
func (tc *TypeChecker) lookupVariableDefinition(variable *ast.VariableNode) (*Symbol, *Scope, error) {
	for scope := tc.currentScope; scope != nil; scope = scope.Parent {
		if sym, exists := scope.Symbols[variable.Ident.Id]; exists {
			return &sym, scope, nil
		}
	}
	return nil, nil, fmt.Errorf("undefined variable: %s", variable.Ident.String())
}

// lookupVariableType finds a variable's type in the current scope chain
func (tc *TypeChecker) LookupVariableType(variable *ast.VariableNode) (ast.TypeNode, error) {
	symbol, err := tc.lookupSymbol(variable.Ident.Id)
	if err != nil {
		return ast.TypeNode{}, err
	}
	return symbol.Type, nil
}

func (tc *TypeChecker) lookupSymbol(ident ast.Identifier) (*Symbol, error) {
	for scope := tc.currentScope; scope != nil; scope = scope.Parent {
		if sym, exists := scope.Symbols[ident]; exists {
			return &sym, nil
		}
	}
	return nil, fmt.Errorf("undefined symbol: %s", ident)
}

func (tc *TypeChecker) LookupFunctionReturnType(function *ast.FunctionNode) (ast.TypeNode, error) {
	symbol, err := tc.lookupSymbol(function.Ident.Id)
	if err != nil {
		return ast.TypeNode{}, err
	}
	if symbol.Kind != SymbolFunction {
		return ast.TypeNode{}, fmt.Errorf("%s is not a function", function.Ident)
	}
	return symbol.Type, nil
}

func (tc *TypeChecker) LookupAssertionType(assertion *ast.AssertionNode) (ast.TypeNode, error) {
	signature := tc.Types[tc.hasher.Hash(assertion)]
	return signature, nil
}
