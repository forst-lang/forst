package typechecker

import (
	"fmt"
	"forst/pkg/ast"
)

// inferFunctionReturnType infers the return type of a function from its body
func (tc *TypeChecker) inferFunctionReturnType(body []ast.Node) (ast.TypeNode, error) {
	// For empty functions, default to void return type
	if len(body) == 0 {
		return ast.TypeNode{Name: ast.TypeVoid}, nil
	}

	// Get last statement which should be the implicit return value
	lastStmt := body[len(body)-1]

	// If last statement is an expression, its type is the return type
	if expr, ok := lastStmt.(ast.ExpressionNode); ok {
		return tc.inferExpressionType(expr)
	}

	// If no expression found, default to void
	return ast.TypeNode{Name: ast.TypeVoid}, nil
}

// inferTypes handles type inference for a single node
func (tc *TypeChecker) inferTypes(node ast.Node) (*ast.TypeNode, error) {
	// Check if we've already inferred this node's type
	hash := tc.hasher.Hash(node)
	if existingType, exists := tc.Types[hash]; exists {
		return &existingType, nil
	}

	switch n := node.(type) {
	case ast.ImportNode:
		return nil, nil
	case ast.ImportGroupNode:
		return nil, nil
	case ast.PackageNode:
		return nil, nil
	case ast.FunctionNode:
		if n.ExplicitReturnType.IsExplicit() {
			tc.storeType(node, n.ExplicitReturnType)
			return &n.ExplicitReturnType, nil
		}
		inferredType, err := tc.inferFunctionReturnType(n.Body)
		if err != nil {
			return nil, err
		}
		tc.storeType(node, inferredType)
		tc.storeInferredFunctionReturnType(&n, inferredType)
		return &inferredType, nil
	case ast.ExpressionNode:
		inferredType, err := tc.inferExpressionType(n)
		if err != nil {
			return nil, err
		}
		tc.storeType(node, inferredType)
		return &inferredType, nil
	}
	panic(typecheckErrorMessageWithNode(&node, "unsupported node type"))
}

func (tc *TypeChecker) inferExpressionType(expr ast.Node) (ast.TypeNode, error) {
	switch e := expr.(type) {
	case ast.BinaryExpressionNode:
		return tc.unifyTypes(e.Left, e.Right, e.Operator)

	case ast.IntLiteralNode:
		return ast.TypeNode{Name: ast.TypeInt}, nil

	case ast.FloatLiteralNode:
		return ast.TypeNode{Name: ast.TypeFloat}, nil

	case ast.StringLiteralNode:
		return ast.TypeNode{Name: ast.TypeString}, nil

	case ast.BoolLiteralNode:
		return ast.TypeNode{Name: ast.TypeBool}, nil

	case ast.VariableNode:
		return tc.LookupVariableType(&e)

	case ast.FunctionCallNode:
		if signature, exists := tc.Functions[e.Function.Id]; exists {
			return signature.ReturnType, nil
		}
		return ast.TypeNode{}, fmt.Errorf("undefined function: %s", e.Function)

	default:
		return ast.TypeNode{}, fmt.Errorf("cannot infer type for expression: %T", expr)
	}
}

// unifyTypes attempts to unify two types based on the operator and operand types
func (tc *TypeChecker) unifyTypes(left ast.Node, right ast.Node, operator ast.TokenType) (ast.TypeNode, error) {
	leftType, err := tc.inferExpressionType(left)
	if err != nil {
		return ast.TypeNode{}, err
	}

	rightType, err := tc.inferExpressionType(right)
	if err != nil {
		return ast.TypeNode{}, err
	}

	// Check type compatibility and determine result type
	if operator.IsArithmeticBinaryOperator() {
		if leftType.Name != rightType.Name {
			return ast.TypeNode{}, fmt.Errorf("type mismatch in arithmetic expression: %s and %s",
				leftType.Name, rightType.Name)
		}
		return leftType, nil

	} else if operator.IsComparisonBinaryOperator() {
		if leftType.Name != rightType.Name {
			return ast.TypeNode{}, fmt.Errorf("type mismatch in comparison expression: %s and %s",
				leftType.Name, rightType.Name)
		}
		return ast.TypeNode{Name: ast.TypeBool}, nil

	} else if operator.IsLogicalBinaryOperator() {
		if leftType.Name != rightType.Name {
			return ast.TypeNode{}, fmt.Errorf("type mismatch in logical expression: %s and %s",
				leftType.Name, rightType.Name)
		}
		return ast.TypeNode{Name: ast.TypeBool}, nil
	}

	panic(typecheckError("unsupported operator"))
}
