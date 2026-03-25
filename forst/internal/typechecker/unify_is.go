package typechecker

import (
	"fmt"
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// getLeftmostVariable returns the leftmost variable in an expression
func (tc *TypeChecker) getLeftmostVariable(node ast.Node) (ast.Node, error) {
	switch n := node.(type) {
	case ast.VariableNode:
		return n, nil
	case ast.BinaryExpressionNode:
		return tc.getLeftmostVariable(n.Left)
	default:
		return nil, fmt.Errorf("expression does not start with a variable: %T", node)
	}
}

// unifyIsOperator handles type unification for the 'is' operator
func (tc *TypeChecker) unifyIsOperator(left ast.Node, right ast.Node, leftType ast.TypeNode, rightType ast.TypeNode) (ast.TypeNode, error) {
	// Get the leftmost variable to check against type guard receiver
	leftmostVar, err := tc.getLeftmostVariable(left)
	if err != nil {
		return ast.TypeNode{}, fmt.Errorf("invalid left-hand side of 'is' operator: %v", err)
	}

	// Get inferred types for the leftmost variable
	varLeftTypes, err := tc.inferExpressionType(leftmostVar)
	if err != nil {
		return ast.TypeNode{}, fmt.Errorf("failed to infer type of leftmost variable: %v", err)
	}
	if len(varLeftTypes) == 0 {
		return ast.TypeNode{}, fmt.Errorf("leftmost variable in IS expression %s has an empty type", leftmostVar.(ast.VariableNode).Ident.ID)
	}
	varLeftType := varLeftTypes[0]

	// Handle different right-hand side types
	switch r := right.(type) {
	case ast.TypeDefAssertionExpr:
		if err := tc.validateTypeDefAssertion(r.Assertion, varLeftType); err != nil {
			return ast.TypeNode{}, err
		}
		// Process type guard fields
		tc.processTypeGuardFields(&ast.ShapeNode{}, r.Assertion)

	case ast.ShapeNode:
		tc.log.WithFields(logrus.Fields{
			"leftType":    varLeftType.Ident,
			"rightType":   "Shape",
			"shapeFields": fmt.Sprintf("%v", r.Fields),
		}).Debugf("Starting shape comparison")

		leftShapeFields, err := tc.getShapeFields(varLeftType, leftmostVar)
		if err != nil {
			return ast.TypeNode{}, err
		}

		if err := tc.ValidateShapeFields(r, leftShapeFields, varLeftType.Ident); err != nil {
			return ast.TypeNode{}, err
		}

	case ast.AssertionNode:
		if err := tc.validateAssertionNode(r, varLeftType); err != nil {
			return ast.TypeNode{}, err
		}

	default:
		if rightType.Ident != ast.TypeShape {
			return ast.TypeNode{}, fmt.Errorf("right-hand side of 'is' must be a Shape type or assertion, got %s", rightType.Ident)
		}
	}

	return ast.TypeNode{Ident: ast.TypeBool}, nil
}
