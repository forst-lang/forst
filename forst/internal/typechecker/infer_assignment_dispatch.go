package typechecker

import (
	"fmt"

	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferAssignmentTypes(assign ast.AssignmentNode) error {
	tc.log.WithFields(logrus.Fields{
		"assignment": assign.String(),
		"lvalues":    len(assign.LValues),
		"rvalues":    len(assign.RValues),
		"function":   "inferAssignmentTypes",
	}).Trace("Starting type inference for assignment")

	if assign.CompoundOp != "" {
		return tc.inferCompoundAssignmentEntry(assign)
	}

	resolvedTypes, err := tc.resolveAssignmentRValueTypes(assign)
	if err != nil {
		return err
	}
	resolvedTypes = tc.distributeAssignmentReturnTypes(assign, resolvedTypes)

	if err := tc.checkAssignmentMapCommaOk(assign); err != nil {
		return err
	}
	if len(assign.RValues) > 0 && len(resolvedTypes) != len(assign.LValues) {
		return reportf(spanOfNode(assign), "assignment-arity",
			"assignment arity mismatch",
			fmt.Sprintf("%d left-hand value(s) but the right-hand side produces %d value(s).", len(assign.LValues), len(resolvedTypes)),
			"match the number of names to the return values or split the assignment")
	}

	if tc.log != nil {
		tc.log.WithFields(logrus.Fields{
			"resolvedTypes": resolvedTypes,
		}).Debug("After distributing function return types to LValues")
	}

	if err := tc.inferAssignmentLValues(assign, resolvedTypes); err != nil {
		return err
	}

	tc.log.WithFields(logrus.Fields{
		"assignment":    assign.String(),
		"lvalues":       assign.LValues,
		"resolvedTypes": resolvedTypes,
		"function":      "inferAssignmentTypes",
	}).Trace("Finished type inference for assignment")

	tc.bindVariableGoTypesFromCall(assign)
	tc.recordAssignmentAliases(assign, resolvedTypes)
	tc.applyAssignmentWriteInvalidation(assign)
	return nil
}

func (tc *TypeChecker) inferCompoundAssignmentEntry(assign ast.AssignmentNode) error {
	asSpan := spanOfNode(assign)
	if assign.IsShort {
		return reportf(asSpan, "compound-assign-short",
			"compound assignment cannot use `:=",
			"Compound operators like `+=` require an existing variable (`=`).",
			"declare the variable first, then use `=` with `+=` / `-=` / …")
	}
	if len(assign.LValues) != 1 || len(assign.RValues) != 1 {
		return reportf(asSpan, "compound-assign-arity",
			"compound assignment needs one value on each side",
			"Compound assignment requires exactly one left- and right-hand side.",
			"split into separate statements or use a single target")
	}
	for _, et := range assign.ExplicitTypes {
		if et != nil {
			return reportf(asSpan, "compound-assign-explicit-type",
				"compound assignment does not support explicit types",
				"Left-hand explicit types are not allowed with `+=`, `-=`, etc.",
				"drop the type annotation or use plain `=` assignment")
		}
	}
	return tc.inferCompoundAssignmentTypes(assign)
}

func (tc *TypeChecker) distributeAssignmentReturnTypes(assign ast.AssignmentNode, resolvedTypes [][]ast.TypeNode) [][]ast.TypeNode {
	if tc.log != nil {
		tc.log.WithFields(logrus.Fields{
			"resolvedTypes": resolvedTypes,
			"LValues":       assign.LValues,
			"RValues":       assign.RValues,
		}).Debug("Before distributing function return types to LValues")
	}
	if len(assign.RValues) == 1 && len(resolvedTypes) == 1 && len(assign.LValues) > 1 {
		switch assign.RValues[0].(type) {
		case ast.FunctionCallNode, ast.MethodCallNode:
			if len(resolvedTypes[0]) == len(assign.LValues) {
				newResolved := make([][]ast.TypeNode, len(assign.LValues))
				for i := range assign.LValues {
					newResolved[i] = []ast.TypeNode{resolvedTypes[0][i]}
				}
				return newResolved
			}
			if len(resolvedTypes[0]) == 1 && len(assign.LValues) == 2 &&
				resolvedTypes[0][0].IsResultType() && len(resolvedTypes[0][0].TypeParams) >= 2 {
				rt := resolvedTypes[0][0]
				return [][]ast.TypeNode{
					{rt.TypeParams[0]},
					{rt.TypeParams[1]},
				}
			}
			if len(resolvedTypes[0]) == 1 && resolvedTypes[0][0].IsTupleType() &&
				len(resolvedTypes[0][0].TypeParams) == len(assign.LValues) {
				tp := resolvedTypes[0][0].TypeParams
				newResolved := make([][]ast.TypeNode, len(assign.LValues))
				for i := range assign.LValues {
					newResolved[i] = []ast.TypeNode{tp[i]}
				}
				return newResolved
			}
		}
	}
	return resolvedTypes
}

func (tc *TypeChecker) checkAssignmentMapCommaOk(assign ast.AssignmentNode) error {
	if len(assign.RValues) == 1 && len(assign.LValues) == 2 {
		if idx, ok := assign.RValues[0].(ast.IndexExpressionNode); ok {
			targetTypes, err := tc.inferExpressionType(idx.Target)
			if err != nil {
				return err
			}
			if len(targetTypes) == 1 && targetTypes[0].Ident == ast.TypeMap {
				return reportf(spanIndexExpr(idx), "map-comma-ok-unsupported",
					"map comma-ok assignment is not supported",
					"Forst does not support Go's `(v, ok := m[k])` map comma-ok form.",
					"use a single-value map read and handle missing keys with Result or explicit checks")
			}
		}
	}
	return nil
}

func (tc *TypeChecker) recordAssignmentAliases(assign ast.AssignmentNode, resolvedTypes [][]ast.TypeNode) {
	for i, lv := range assign.LValues {
		var rhs ast.ExpressionNode
		if i < len(assign.RValues) {
			rhs = assign.RValues[i]
		} else if len(assign.RValues) == 1 {
			rhs = assign.RValues[0]
		}
		if rhs == nil {
			continue
		}
		var rhsTypes []ast.TypeNode
		if i < len(resolvedTypes) {
			rhsTypes = resolvedTypes[i]
		}
		tc.recordAssignmentAlias(lv, rhs, rhsTypes)
		if lit, ok := rhs.(ast.FunctionLiteralNode); ok {
			tc.bindClosureCaptures(lv, lit)
		}
	}
}

func (tc *TypeChecker) applyAssignmentWriteInvalidation(assign ast.AssignmentNode) {
	if !assign.IsShort {
		for _, lv := range assign.LValues {
			writePath, span := tc.writePathFromAssignTarget(lv)
			if writePath != nil {
				tc.applyWriteInvalidation(writePath, span)
			}
		}
		return
	}
	for i := range assign.LValues {
		if i < len(assign.RValues) {
			if _, ok := assign.RValues[i].(ast.FunctionLiteralNode); ok {
				continue
			}
		}
	}
}
