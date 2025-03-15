package typechecker

import (
	"fmt"

	"forst/pkg/ast"
)

// lookupVariableType finds a variable's type in the current scope chain
func (tc *TypeChecker) LookupVariableType(variable *ast.VariableNode) (ast.TypeNode, error) {
	scope := tc.currentScope
	for scope != nil {
		if _, exists := scope.Variables[variable.Ident.Name]; exists {
			if typ, ok := tc.Types[tc.hasher.Hash(variable)]; ok {
				return typ, nil
			}
		}
		scope = scope.parent
	}
	return ast.TypeNode{}, fmt.Errorf("undefined variable: %s", variable.Ident.Name)
}

func (tc *TypeChecker) LookupFunctionReturnType(function *ast.FunctionNode) (ast.TypeNode, error) {
	signature := tc.Functions[&function.Ident]
	return signature.ReturnType, nil
}
