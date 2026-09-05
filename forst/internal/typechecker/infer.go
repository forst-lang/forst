package typechecker

import (
	"forst/internal/ast"
)

// inferNodeTypes handles type inference for a list of nodes
func (tc *TypeChecker) inferNodeTypes(nodes []ast.Node, scopeNode ast.Node) ([][]ast.TypeNode, error) {
	if err := tc.RestoreScope(scopeNode); err != nil {
		return nil, err
	}
	inferredTypes := make([][]ast.TypeNode, len(nodes))
	for i, node := range nodes {
		inferredType, err := tc.inferNodeType(node)
		if err != nil {
			return nil, err
		}

		inferredTypes[i] = inferredType
	}
	return inferredTypes, nil
}
