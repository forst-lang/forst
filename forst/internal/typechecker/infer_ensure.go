package typechecker

import (
	"forst/internal/ast"
)

func (tc *TypeChecker) inferEnsureType(ensure ast.EnsureNode) (any, error) {
	variableType, err := tc.LookupVariableType(&ensure.Variable, tc.scopeStack.CurrentScope())
	if err != nil {
		return nil, err
	}

	// Store the base type of the assertion's variable
	tc.storeInferredType(ensure.Assertion, []ast.TypeNode{variableType})

	if ensure.Error != nil {
		return nil, nil
	}

	return nil, nil
}
