package typechecker

import (
	"fmt"
	"forst/pkg/ast"
)

func failWithTypeMismatch(fn ast.FunctionNode, inferred ast.TypeNode, parsed ast.TypeNode) error {
	return fmt.Errorf("inferred return type %s of function %s does not match parsed return type %s", inferred.String(), fn.Id(), parsed.String())
}

// Ensures that the first type matches the expected type, otherwise returns an error
func ensureMatching(fn ast.FunctionNode, actual ast.TypeNode, expected ast.TypeNode) (ast.TypeNode, error) {
	if expected.IsImplicit() {
		// If the expected type is implicit, we have nothing to check against
		return actual, nil
	}

	if actual.Name != expected.Name {
		return actual, failWithTypeMismatch(fn, actual, expected)
	}

	return actual, nil
}

// inferFunctionReturnType infers the return type of a function from its body
func (tc *TypeChecker) inferFunctionReturnType(fn ast.FunctionNode) (ast.TypeNode, error) {
	parsedType := fn.ReturnType
	inferredType := ast.TypeNode{Name: ast.TypeVoid}

	// For empty functions, default to void return type
	if len(fn.Body) == 0 {
		return ensureMatching(fn, inferredType, parsedType)
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
				return inferredType, failWithTypeMismatch(fn, inferredType, firstType)
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
			return inferredType, failWithTypeMismatch(fn, inferredType, exprType)
		}

		return ensureMatching(fn, inferredType, parsedType)
	}

	// If we found return statements, use the first return type
	if len(returnStmtTypes) > 0 {
		inferredType = returnStmtTypes[0]

		return ensureMatching(fn, inferredType, parsedType)
	}

	return ensureMatching(fn, inferredType, parsedType)
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

func (tc *TypeChecker) inferEnsureType(ensure ast.EnsureNode) (*ast.TypeNode, error) {
	variableType, err := tc.LookupVariableType(&ensure.Variable)
	if err != nil {
		return nil, err
	}

	// Store the base type of the assertion's variable
	tc.storeInferredType(ensure.Assertion, variableType)

	if ensure.Error != nil {
		return nil, nil
	}

	return nil, nil
}

// inferNodeTypes handles type inference for a list of nodes
func (tc *TypeChecker) inferNodeTypes(nodes []ast.Node) ([]*ast.TypeNode, error) {
	inferredTypes := make([]*ast.TypeNode, len(nodes))
	for i, node := range nodes {
		inferredType, err := tc.inferNodeType(node)
		if err != nil {
			return nil, err
		}

		inferredTypes[i] = inferredType
	}
	return inferredTypes, nil
}

// inferNodeType handles type inference for a single node
func (tc *TypeChecker) inferNodeType(node ast.Node) (*ast.TypeNode, error) {
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
		tc.pushScope(&n)

		inferredType, err := tc.inferFunctionReturnType(n)
		if err != nil {
			return nil, err
		}
		tc.storeInferredFunctionReturnType(&n, inferredType)

		_, err = tc.inferNodeTypes(n.Body)
		if err != nil {
			return nil, err
		}

		tc.popScope()

		return &inferredType, nil
	case ast.ExpressionNode:
		inferredType, err := tc.inferExpressionType(n)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(node, inferredType)
		return &inferredType, nil
	case ast.EnsureNode:
		inferredType, err := tc.inferEnsureType(n)
		if err != nil {
			return nil, err
		}

		if n.Block != nil {
			tc.pushScope(n.Block)
			_, err = tc.inferNodeTypes(n.Block.Body)
			if err != nil {
				return nil, err
			}
			tc.popScope()
		}

		return inferredType, nil
	case ast.AssignmentNode:
		if err := tc.inferAssignmentTypes(n); err != nil {
			return nil, err
		}
		return nil, nil
	}
	panic(typecheckErrorMessageWithNode(&node, fmt.Sprintf("unsupported node type %T", node)))
}

func (tc *TypeChecker) inferExpressionType(expr ast.Node) (ast.TypeNode, error) {
	switch e := expr.(type) {
	case ast.BinaryExpressionNode:
		inferredType, err := tc.unifyTypes(e.Left, e.Right, e.Operator)
		if err != nil {
			return ast.TypeNode{}, err
		}
		tc.storeInferredType(e, inferredType)
		return inferredType, nil

	case ast.UnaryExpressionNode:
		inferredType, err := tc.unifyTypes(e.Operand, nil, e.Operator)
		if err != nil {
			return ast.TypeNode{}, err
		}
		tc.storeInferredType(e, inferredType)
		return inferredType, nil

	case ast.IntLiteralNode:
		typ := ast.TypeNode{Name: ast.TypeInt}
		tc.storeInferredType(e, typ)
		return typ, nil

	case ast.FloatLiteralNode:
		typ := ast.TypeNode{Name: ast.TypeFloat}
		tc.storeInferredType(e, typ)
		return typ, nil

	case ast.StringLiteralNode:
		typ := ast.TypeNode{Name: ast.TypeString}
		tc.storeInferredType(e, typ)
		return typ, nil

	case ast.BoolLiteralNode:
		typ := ast.TypeNode{Name: ast.TypeBool}
		tc.storeInferredType(e, typ)
		return typ, nil

	case ast.VariableNode:
		// Look up the variable's type and store it for this node
		typ, err := tc.LookupVariableType(&e)
		if err != nil {
			return ast.TypeNode{}, err
		}
		tc.storeInferredType(e, typ)
		return typ, nil

	case ast.FunctionCallNode:
		if signature, exists := tc.Functions[e.Function.Id]; exists {
			tc.storeInferredType(e, signature.ReturnType)
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

func (tc *TypeChecker) storeInferredVariableType(variable ast.VariableNode, typ ast.TypeNode) {
	tc.storeSymbol(variable.Ident.Id, typ, SymbolVariable)
	tc.storeInferredType(variable, typ)
}

func (tc *TypeChecker) registerFunction(fn ast.FunctionNode) {
	// Store function signature
	params := make([]ParameterSignature, len(fn.Params))
	for i, param := range fn.Params {
		params[i] = ParameterSignature{
			Ident: param.Ident,
			Type:  param.Type,
		}
	}
	tc.Functions[fn.Id()] = FunctionSignature{
		Ident:      fn.Ident,
		Parameters: params,
		ReturnType: fn.ReturnType,
	}

	// Store function symbol
	tc.storeSymbol(fn.Ident.Id, fn.ReturnType, SymbolFunction)

	// Store parameter symbols
	for _, param := range fn.Params {
		tc.storeSymbol(param.Ident.Id, param.Type, SymbolParameter)
	}
}

func (tc *TypeChecker) inferAssignmentTypes(assign ast.AssignmentNode) error {
	for i, value := range assign.RValues {
		inferredType, err := tc.inferExpressionType(value)
		if err != nil {
			return err
		}

		// Store the inferred type for each identifier
		if i < len(assign.LValues) {
			variableNode := assign.LValues[i]
			tc.storeInferredVariableType(variableNode, inferredType)
			// Also store the type for the variable node itself
			tc.storeInferredType(variableNode, inferredType)
		}
	}
	return nil
}
