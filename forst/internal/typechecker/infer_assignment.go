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

	// Collect types for each RValue
	resolvedTypes = make([][]ast.TypeNode, 0, len(assign.RValues))
	for _, rvalue := range assign.RValues {
		types, err := tc.inferExpressionType(rvalue)
		if err != nil {
			return err
		}
		resolvedTypes = append(resolvedTypes, types)
	}

	// Debug: print resolvedTypes, LValues, and RValues before distribution
	if tc.log != nil {
		tc.log.WithFields(logrus.Fields{
			"resolvedTypes": resolvedTypes,
			"LValues":       assign.LValues,
			"RValues":       assign.RValues,
		}).Debug("Before distributing function return types to LValues")
	}

	// Distribute function call return values to LValues
	if len(assign.RValues) == 1 && len(resolvedTypes) == 1 && len(assign.LValues) > 1 {
		// Check if the single RValue is a function call and its return types match LValues
		if _, ok := assign.RValues[0].(ast.FunctionCallNode); ok {
			if len(resolvedTypes[0]) == len(assign.LValues) {
				newResolved := make([][]ast.TypeNode, len(assign.LValues))
				for i := range assign.LValues {
					newResolved[i] = []ast.TypeNode{resolvedTypes[0][i]}
				}
				resolvedTypes = newResolved
			}
		}
	}

	// Debug: print resolvedTypes after distribution
	if tc.log != nil {
		tc.log.WithFields(logrus.Fields{
			"resolvedTypes": resolvedTypes,
		}).Debug("After distributing function return types to LValues")
	}

	// Store types for each LValue
	for i, variableNode := range assign.LValues {
		// Check if this is a var declaration (has explicit types) or a regular assignment
		isVarDeclaration := len(assign.ExplicitTypes) > 0 && assign.ExplicitTypes[i] != nil

		// For regular assignments (not var declarations), check if variable is declared
		if !assign.IsShort && !isVarDeclaration {
			_, exists := tc.scopeStack.LookupVariableType(variableNode.Ident.ID)
			if !exists {
				return fmt.Errorf("assignment to undeclared variable '%s' is not allowed; use 'var' or ':='", variableNode.Ident.ID)
			}
		}

		// For var declarations, check that the explicit type is defined
		if isVarDeclaration {
			explicitType := variableNode.ExplicitType
			isPointer := explicitType.Ident == ast.TypePointer
			isBuiltin := explicitType.Ident == ast.TypeString || explicitType.Ident == ast.TypeInt || explicitType.Ident == ast.TypeFloat || explicitType.Ident == ast.TypeBool || explicitType.Ident == ast.TypeError || explicitType.Ident == ast.TypeVoid || explicitType.Ident == ast.TypeArray || explicitType.Ident == ast.TypeMap || explicitType.Ident == ast.TypeShape || explicitType.Ident == ast.TypeObject
			_, isDefined := tc.Defs[explicitType.Ident]
			if !isPointer && !isBuiltin && !isDefined {
				return fmt.Errorf("undefined type name '%s' in variable declaration", explicitType.Ident)
			}
		}

		// For var declarations, check nil assignment is only allowed for pointer, interface, map, slice, channel, or function types
		if isVarDeclaration && len(assign.RValues) == 1 {
			if _, isNil := assign.RValues[0].(ast.NilLiteralNode); isNil {
				explicitType := variableNode.ExplicitType
				isPointer := explicitType.Ident == ast.TypePointer
				isInterface := explicitType.Ident == ast.TypeObject
				isMap := explicitType.Ident == ast.TypeMap
				isArray := explicitType.Ident == ast.TypeArray
				isFunc := explicitType.Ident == ast.TypeIdent("Func") // TODO: update if you have a function type ident
				if !(isPointer || isInterface || isMap || isArray || isFunc) {
					return fmt.Errorf("cannot assign nil to variable of type '%s'", explicitType.Ident)
				}
			}
		}

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
