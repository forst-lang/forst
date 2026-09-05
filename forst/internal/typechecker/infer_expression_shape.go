package typechecker

import (
	"fmt"
	"forst/internal/ast"
)

func (tc *TypeChecker) inferExpressionShapeAssertion(expr ast.Node) ([]ast.TypeNode, bool, error) {
	switch e := expr.(type) {
	case ast.ShapeNode:

		inferredType, err := tc.inferShapeType(e, nil)
		if err != nil {
			return nil, true, fmt.Errorf("failed to infer shape type: %w", err)
		}
		tc.storeInferredType(e, []ast.TypeNode{inferredType})
		return []ast.TypeNode{inferredType}, true, nil
	case ast.AssertionNode:

		inferredType, err := tc.InferAssertionType(&e, false, "", nil)
		if err != nil {
			return nil, true, err
		}
		tc.storeInferredType(e, inferredType)
		return inferredType, true, nil
	}
	return nil, false, nil
}
