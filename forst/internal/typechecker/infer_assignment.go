package typechecker

import (
	"fmt"
	"forst/internal/ast"
)

func (tc *TypeChecker) inferAssignmentTypes(assign ast.AssignmentNode) error {
	var resolvedTypes []ast.TypeNode

	// Collect all resolved types from RValues
	for _, value := range assign.RValues {
		if callExpr, isCall := value.(ast.FunctionCallNode); isCall {
			// Get function signature and check return types
			if sig, exists := tc.Functions[callExpr.Function.ID]; exists {
				resolvedTypes = append(resolvedTypes, sig.ReturnTypes...)
			} else {
				return fmt.Errorf("undefined function: %s", callExpr.Function.ID)
			}
		} else {
			inferredType, err := tc.inferExpressionType(value)
			if err != nil {
				return err
			}
			resolvedTypes = append(resolvedTypes, inferredType...)
		}
	}

	// Check if number of LValues matches total resolved types
	if len(assign.LValues) != len(resolvedTypes) {
		return fmt.Errorf("assignment mismatch: %d variables but got %d values",
			len(assign.LValues), len(resolvedTypes))
	}

	// Store types for each LValue
	for i, variableNode := range assign.LValues {
		// If the variable has an explicit type annotation, use it
		if variableNode.ExplicitType.Ident != "" && variableNode.ExplicitType.Ident != ast.TypeImplicit {
			tc.storeInferredVariableType(variableNode, variableNode.ExplicitType)
			tc.storeInferredType(variableNode, []ast.TypeNode{variableNode.ExplicitType})
		} else {
			tc.storeInferredVariableType(variableNode, resolvedTypes[i])
			tc.storeInferredType(variableNode, []ast.TypeNode{resolvedTypes[i]})
		}
	}

	return nil
}
