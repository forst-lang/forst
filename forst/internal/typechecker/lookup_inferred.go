package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// LookupInferredType looks up the inferred type of a node in the current scope.
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
