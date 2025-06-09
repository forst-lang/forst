package typechecker

import (
	"fmt"
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferAssignmentTypes(assign ast.AssignmentNode) error {
	var resolvedTypes [][]ast.TypeNode

	tc.log.WithFields(logrus.Fields{
		"assignment": assign.String(),
		"lvalues":    len(assign.LValues),
		"rvalues":    len(assign.RValues),
		"function":   "inferAssignmentTypes",
	}).Trace("Starting type inference for assignment")

	// Collect all resolved types from RValues
	for i, value := range assign.RValues {
		tc.log.WithFields(logrus.Fields{
			"index":    i,
			"value":    fmt.Sprintf("%#v", value),
			"function": "inferAssignmentTypes",
		}).Trace("Inferring type for RValue")
		if callExpr, isCall := value.(ast.FunctionCallNode); isCall {
			// Get function signature and check return types
			if sig, exists := tc.Functions[callExpr.Function.ID]; exists {
				tc.log.WithFields(logrus.Fields{
					"fn":          callExpr.Function.ID,
					"function":    "inferAssignmentTypes",
					"returnTypes": sig.ReturnTypes,
				}).Trace("Found function signature for call")
				resolvedTypes = append(resolvedTypes, sig.ReturnTypes)
			} else {
				tc.log.WithFields(logrus.Fields{
					"fn":       callExpr.Function.ID,
					"function": "inferAssignmentTypes",
				}).Error("Undefined function in assignment")
				return fmt.Errorf("undefined function: %s", callExpr.Function.ID)
			}
		} else {
			inferredType, err := tc.inferExpressionType(value)
			if err != nil {
				tc.log.WithFields(logrus.Fields{
					"value":    fmt.Sprintf("%#v", value),
					"error":    err,
					"function": "inferAssignmentTypes",
				}).Error("Failed to infer type for RValue")
				return err
			}
			tc.log.WithFields(logrus.Fields{
				"value":        fmt.Sprintf("%#v", value),
				"inferredType": inferredType,
				"function":     "inferAssignmentTypes",
			}).Trace("Inferred type for RValue")
			resolvedTypes = append(resolvedTypes, inferredType)
		}
	}

	// Check if number of LValues matches total resolved types
	if len(assign.LValues) != len(resolvedTypes) {
		tc.log.WithFields(logrus.Fields{
			"lvalues":       len(assign.LValues),
			"resolvedTypes": len(resolvedTypes),
			"function":      "inferAssignmentTypes",
		}).Error("Assignment mismatch")
		return fmt.Errorf("assignment mismatch: %d variables but got %d values",
			len(assign.LValues), len(resolvedTypes))
	}

	// Store types for each LValue
	for i, variableNode := range assign.LValues {
		if variableNode.ExplicitType.Ident != "" && variableNode.ExplicitType.Ident != ast.TypeImplicit {
			tc.log.WithFields(logrus.Fields{
				"variable":     variableNode.Ident.ID,
				"explicitType": variableNode.ExplicitType.Ident,
				"function":     "inferAssignmentTypes",
			}).Trace("Using explicit type for variable (alias preserved)")
			tc.storeInferredVariableType(variableNode, []ast.TypeNode{variableNode.ExplicitType})
			tc.storeInferredType(variableNode, []ast.TypeNode{variableNode.ExplicitType})
		} else {
			tc.log.WithFields(logrus.Fields{
				"variable":     variableNode.Ident.ID,
				"resolvedType": resolvedTypes[i],
				"function":     "inferAssignmentTypes",
			}).Trace("Using resolved type for variable")
			tc.storeInferredVariableType(variableNode, resolvedTypes[i])
			tc.storeInferredType(variableNode, resolvedTypes[i])
		}
	}

	tc.log.WithFields(logrus.Fields{
		"assignment":    assign.String(),
		"lvalues":       assign.LValues,
		"resolvedTypes": resolvedTypes,
		"function":      "inferAssignmentTypes",
	}).Trace("Finished type inference for assignment")

	return nil
}
