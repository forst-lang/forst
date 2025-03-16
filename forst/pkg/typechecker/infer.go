package typechecker

import (
	"fmt"
	"forst/pkg/ast"
)

func failWithTypeMismatch(inferredType ast.TypeNode, explicitReturnType ast.TypeNode) error {
	return fmt.Errorf("inferred return type %s does not match explicit return type %s", inferredType.String(), explicitReturnType.String())
}

// Ensures that the first type matches the expected type, otherwise returns an error
func ensureMatching(typ ast.TypeNode, expectedType ast.TypeNode) (ast.TypeNode, error) {
	if expectedType.IsImplicit() {
		// If the expected type is implicit, we have nothing to check against
		return typ, nil
	}

	if typ.Name != expectedType.Name {
		return typ, failWithTypeMismatch(typ, expectedType)
	}

	return typ, nil
}

// inferFunctionReturnType infers the return type of a function from its body
func (tc *TypeChecker) inferFunctionReturnType(fn ast.FunctionNode) (ast.TypeNode, error) {
	explicitReturnType := fn.ExplicitReturnType
	inferredType := ast.TypeNode{Name: ast.TypeVoid}

	// For empty functions, default to void return type
	if len(fn.Body) == 0 {
		return ensureMatching(inferredType, explicitReturnType)
	}

	// Find all return statements and collect their types
	returnStmtTypes := make([]ast.TypeNode, 0)
	for _, stmt := range fn.Body {
		if retStmt, ok := stmt.(ast.ReturnNode); ok {
			// Get type of return expression
			retType, err := tc.inferExpressionType(retStmt.Value)
			if err != nil {
				return inferredType, err
			}
			returnStmtTypes = append(returnStmtTypes, retType)
		}
	}

	// If we found return statements, verify they all have the same type
	if len(returnStmtTypes) > 0 {
		firstType := returnStmtTypes[0]
		for _, retType := range returnStmtTypes[1:] {
			if retType.Name != firstType.Name {
				return inferredType, failWithTypeMismatch(inferredType, firstType)
			}
		}
	}

	// Get last statement which should be the implicit return value
	lastStmt := fn.Body[len(fn.Body)-1]

	// If last statement is an expression, its type is the return type
	if expr, ok := lastStmt.(ast.ExpressionNode); ok {
		exprType, err := tc.inferExpressionType(expr)
		if err != nil {
			return ast.TypeNode{}, err
		}

		inferredType = exprType

		// If we found return statements, verify the expression type matches
		if len(returnStmtTypes) > 0 && exprType.Name != returnStmtTypes[0].Name {
			return inferredType, failWithTypeMismatch(inferredType, exprType)
		}

		return ensureMatching(inferredType, explicitReturnType)
	}

	// If we found return statements, use the first return type
	if len(returnStmtTypes) > 0 {
		inferredType = returnStmtTypes[0]

		return ensureMatching(inferredType, explicitReturnType)
	}

	return ensureMatching(inferredType, explicitReturnType)
}

func findAlreadyInferredType(tc *TypeChecker, node ast.Node) (*ast.TypeNode, error) {
	hash := tc.hasher.Hash(node)
	if existingType, exists := tc.Types[hash]; exists {
		// Ignore types that are still marked as implicit, as they are not yet inferred
		if existingType.IsImplicit() {
			return nil, nil
		}
		return &existingType, nil
	}
	return nil, nil
}

// inferTypes handles type inference for a single node
func (tc *TypeChecker) inferTypes(node ast.Node) (*ast.TypeNode, error) {
	// Check if we've already inferred this node's type
	alreadyInferredType, err := findAlreadyInferredType(tc, node)
	if err != nil {
		return nil, err
	}
	if alreadyInferredType != nil {
		return alreadyInferredType, nil
	}

	switch n := node.(type) {
	case ast.ImportNode:
		return nil, nil
	case ast.ImportGroupNode:
		return nil, nil
	case ast.PackageNode:
		return nil, nil
	case ast.FunctionNode:
		fmt.Printf("Inferring function return type %s\n", n.Id())
		inferredType, err := tc.inferFunctionReturnType(n)
		if err != nil {
			return nil, err
		}
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

	case ast.UnaryExpressionNode:
		return tc.unifyTypes(e.Operand, nil, e.Operator)

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
