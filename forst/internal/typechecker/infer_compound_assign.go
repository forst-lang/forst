package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"
)

func (tc *TypeChecker) inferCompoundAssignmentTypes(assign ast.AssignmentNode) error {
	lv := assign.LValues[0]
	rv := assign.RValues[0]
	binOp, ok := ast.CompoundAssignBinaryOp(assign.CompoundOp)
	if !ok {
		return fmt.Errorf("unsupported compound assignment operator %s", assign.CompoundOp)
	}

	switch l := lv.(type) {
	case ast.VariableNode:
		parts := strings.Split(string(l.Ident.ID), ".")
		if len(parts) == 1 {
			if _, exists := tc.scopeStack.LookupVariableType(l.Ident.ID); !exists {
				return fmt.Errorf("assignment to undeclared variable '%s' is not allowed; use 'var' or ':='", l.Ident.ID)
			}
		}
	case ast.IndexExpressionNode, ast.DereferenceNode:
	default:
		return fmt.Errorf("unsupported compound assignment target type: %T", lv)
	}

	rhsTypes, err := tc.inferExpressionType(rv)
	if err != nil {
		return err
	}
	if len(rhsTypes) != 1 {
		return fmt.Errorf("compound assignment: right-hand side must have a single type")
	}

	resultType, err := tc.unifyTypes(lv, rv, binOp)
	if err != nil {
		return err
	}

	lhsTypes, err := tc.inferExpressionType(lv)
	if err != nil {
		return err
	}
	if len(lhsTypes) != 1 {
		return fmt.Errorf("compound assignment: left-hand side must have a single type")
	}
	if !tc.IsTypeCompatible(resultType, lhsTypes[0]) {
		return fmt.Errorf("compound assignment type mismatch: cannot assign %s to %s (expected %s)",
			resultType.Ident, lv.String(), lhsTypes[0].Ident)
	}

	tc.storeInferredType(rv, rhsTypes)
	tc.storeInferredType(lv, lhsTypes)
	return nil
}
