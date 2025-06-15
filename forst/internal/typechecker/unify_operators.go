package typechecker

import (
	"fmt"
	"forst/internal/ast"
)

// unifyTypes attempts to unify two types based on the operator and operand types
func (tc *TypeChecker) unifyTypes(left ast.Node, right ast.Node, operator ast.TokenIdent) (ast.TypeNode, error) {
	leftTypes, err := tc.inferExpressionType(left)
	if err != nil {
		return ast.TypeNode{}, err
	}
	if len(leftTypes) != 1 {
		return ast.TypeNode{}, fmt.Errorf("expected single type but got %d types", len(leftTypes))
	}
	leftType := leftTypes[0]

	rightTypes, err := tc.inferExpressionType(right)
	if err != nil {
		return ast.TypeNode{}, err
	}
	if len(rightTypes) != 1 {
		return ast.TypeNode{}, fmt.Errorf("expected single type but got %d types", len(rightTypes))
	}
	rightType := rightTypes[0]

	// Check type compatibility and determine result type
	if operator.IsArithmeticBinaryOperator() {
		return tc.unifyArithmeticOperator(leftType, rightType)
	} else if operator.IsComparisonBinaryOperator() {
		return tc.unifyComparisonOperator(leftType, rightType)
	} else if operator.IsLogicalBinaryOperator() {
		return tc.unifyLogicalOperator(leftType, rightType)
	} else if operator == ast.TokenIs {
		return tc.unifyIsOperator(left, right, leftType, rightType)
	}

	panic(typecheckError("unsupported operator"))
}

// unifyArithmeticOperator handles type unification for arithmetic operators
func (tc *TypeChecker) unifyArithmeticOperator(leftType, rightType ast.TypeNode) (ast.TypeNode, error) {
	if leftType.Ident != rightType.Ident {
		return ast.TypeNode{}, fmt.Errorf("type mismatch in arithmetic expression: %s and %s",
			leftType.Ident, rightType.Ident)
	}
	return leftType, nil
}

// unifyComparisonOperator handles type unification for comparison operators
func (tc *TypeChecker) unifyComparisonOperator(leftType, rightType ast.TypeNode) (ast.TypeNode, error) {
	if leftType.Ident != rightType.Ident {
		return ast.TypeNode{}, fmt.Errorf("type mismatch in comparison expression: %s and %s",
			leftType.Ident, rightType.Ident)
	}
	return ast.TypeNode{Ident: ast.TypeBool}, nil
}

// unifyLogicalOperator handles type unification for logical operators
func (tc *TypeChecker) unifyLogicalOperator(leftType, rightType ast.TypeNode) (ast.TypeNode, error) {
	if leftType.Ident != rightType.Ident {
		return ast.TypeNode{}, fmt.Errorf("type mismatch in logical expression: %s and %s",
			leftType.Ident, rightType.Ident)
	}
	return ast.TypeNode{Ident: ast.TypeBool}, nil
}
