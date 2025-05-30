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
		if leftType.Ident != rightType.Ident {
			return ast.TypeNode{}, fmt.Errorf("type mismatch in arithmetic expression: %s and %s",
				leftType.Ident, rightType.Ident)
		}
		return leftType, nil

	} else if operator.IsComparisonBinaryOperator() {
		if leftType.Ident != rightType.Ident {
			return ast.TypeNode{}, fmt.Errorf("type mismatch in comparison expression: %s and %s",
				leftType.Ident, rightType.Ident)
		}
		return ast.TypeNode{Ident: ast.TypeBool}, nil

	} else if operator.IsLogicalBinaryOperator() {
		if leftType.Ident != rightType.Ident {
			return ast.TypeNode{}, fmt.Errorf("type mismatch in logical expression: %s and %s",
				leftType.Ident, rightType.Ident)
		}
		return ast.TypeNode{Ident: ast.TypeBool}, nil
	}

	panic(typecheckError("unsupported operator"))
}
