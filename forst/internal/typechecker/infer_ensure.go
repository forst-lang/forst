package typechecker

import (
	"forst/internal/ast"
)

func (tc *TypeChecker) inferEnsureType(ensure ast.EnsureNode) (ast.TypeNode, error) {
	variableType, err := tc.LookupVariableType(&ensure.Variable, tc.CurrentScope())
	if err != nil {
		return ast.TypeNode{}, err
	}

	if err := tc.validateAssertionNode(ensure.Assertion, variableType); err != nil {
		return ast.TypeNode{}, err
	}

	// Assertion hover type is stored in infer.go after successor narrowing so `tc.Types` matches the
	// same inference order as `if x is <assertion>` (see applyEnsureSuccessorNarrowing).
	return variableType, nil
}
