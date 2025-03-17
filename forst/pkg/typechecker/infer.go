package typechecker

import (
	"fmt"
	"forst/pkg/ast"
	"strings"

	log "github.com/sirupsen/logrus"
)

func formatTypeList(types []ast.TypeNode) string {
	formatted := make([]string, len(types))
	for i, typ := range types {
		formatted[i] = typ.String()
	}
	return strings.Join(formatted, ", ")
}

func failWithTypeMismatch(fn ast.FunctionNode, inferred []ast.TypeNode, parsed []ast.TypeNode) error {
	return fmt.Errorf("inferred return type %v of function %s does not match parsed return type %v", formatTypeList(inferred), fn.Id(), formatTypeList(parsed))
}

// Ensures that the first type matches the expected type, otherwise returns an error
func ensureMatching(fn ast.FunctionNode, actual []ast.TypeNode, expected []ast.TypeNode) ([]ast.TypeNode, error) {
	if len(expected) == 0 {
		// If the expected type is implicit, we have nothing to check against
		return actual, nil
	}

	if len(actual) != len(expected) {
		return actual, failWithTypeMismatch(fn, actual, expected)
	}

	for i := range expected {
		if actual[i].Name != expected[i].Name {
			return actual, failWithTypeMismatch(fn, actual, expected)
		}
	}

	return actual, nil
}

// inferFunctionReturnType infers the return type of a function from its body
func (tc *TypeChecker) inferFunctionReturnType(fn ast.FunctionNode) ([]ast.TypeNode, error) {
	parsedType := fn.ReturnTypes
	inferredType := []ast.TypeNode{}

	// For empty functions, default to void return type
	if len(fn.Body) == 0 {
		return ensureMatching(fn, inferredType, parsedType)
	}

	// Find all return statements and collect their types
	returnStmtTypes := make([][]ast.TypeNode, 0)
	for _, stmt := range fn.Body {
		if retStmt, ok := stmt.(ast.ReturnNode); ok {
			// Get type of return expression
			retType, err := tc.inferExpressionType(retStmt.Value)
			if err != nil {
				return nil, err
			}
			returnStmtTypes = append(returnStmtTypes, retType)
		}
	}

	// If we found return statements, verify they all have the same type
	if len(returnStmtTypes) > 0 {
		firstType := returnStmtTypes[0]
		for _, retTypes := range returnStmtTypes[1:] {
			for _, retType := range retTypes {
				if retType.Name != firstType[0].Name {
					return nil, failWithTypeMismatch(fn, inferredType, firstType)
				}
			}
		}
	}

	// Get last statement which should be the implicit return value
	lastStmt := fn.Body[len(fn.Body)-1]

	// If last statement is an expression, its type is the return type
	if expr, ok := lastStmt.(ast.ExpressionNode); ok {
		exprTypes, err := tc.inferExpressionType(expr)
		if err != nil {
			return nil, err
		}

		inferredType = exprTypes

		// If we found return statements, verify the expression type matches
		if len(returnStmtTypes) > 0 {
			for i, exprType := range exprTypes {
				if exprType.Name != returnStmtTypes[0][i].Name {
					return nil, failWithTypeMismatch(fn, inferredType, exprTypes)
				}
			}
		}

		return ensureMatching(fn, inferredType, parsedType)
	}

	// If we found return statements, use the first return type
	if len(returnStmtTypes) > 0 {
		inferredType = returnStmtTypes[0]
	}

	// Check if there are any ensure nodes and verify their types match
	for _, stmt := range fn.Body {
		if _, ok := stmt.(ast.EnsureNode); ok {
			if len(inferredType) == 0 {
				inferredType = []ast.TypeNode{
					{Name: ast.TypeError},
				}
			} else {
				if len(inferredType) != 1 && len(inferredType) != 2 {
					return nil, fmt.Errorf("ensure statements require the function to return an error or a tuple with an error as the second type, got %s", formatTypeList(inferredType))
				}

				if inferredType[len(inferredType)-1].Name != ast.TypeError {
					return nil, fmt.Errorf("ensure statements require the function to an error as the second return type, got %s", inferredType[1].Name)
				}
			}
		}
	}

	if len(inferredType) == 0 {
		inferredType = []ast.TypeNode{{Name: ast.TypeVoid}}
	}

	return ensureMatching(fn, inferredType, parsedType)
}

func findAlreadyInferredType(tc *TypeChecker, node ast.Node) ([]ast.TypeNode, error) {
	hash := tc.Hasher.Hash(node)
	if existingType, exists := tc.Types[hash]; exists {
		// Ignore types that are still marked as implicit, as they are not yet inferred
		if len(existingType) > 0 {
			return nil, nil
		}
		return existingType, nil
	}
	return nil, nil
}

func (tc *TypeChecker) inferEnsureType(ensure ast.EnsureNode) (any, error) {
	variableType, err := tc.LookupVariableType(&ensure.Variable, tc.currentScope)
	if err != nil {
		return nil, err
	}

	// Store the base type of the assertion's variable
	tc.storeInferredType(ensure.Assertion, []ast.TypeNode{variableType})

	if ensure.Error != nil {
		return nil, nil
	}

	return nil, nil
}

// inferNodeTypes handles type inference for a list of nodes
func (tc *TypeChecker) inferNodeTypes(nodes []ast.Node) ([][]ast.TypeNode, error) {
	log.Trace("inferNodeTypes", nodes)
	inferredTypes := make([][]ast.TypeNode, len(nodes))
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
func (tc *TypeChecker) inferNodeType(node ast.Node) ([]ast.TypeNode, error) {
	log.Trace("inferNodeType", node)
	// Check if we've already inferred this node's type
	alreadyInferredType, err := findAlreadyInferredType(tc, node)
	if err != nil {
		return nil, err
	}
	if alreadyInferredType != nil {
		// fmt.Println("inferNodeType", node, "already inferred", alreadyInferredType)
		// return alreadyInferredType, nil
	}

	switch n := node.(type) {
	case ast.ImportNode:
		return nil, nil
	case ast.ImportGroupNode:
		return nil, nil
	case ast.PackageNode:
		return nil, nil
	case ast.FunctionNode:
		tc.pushScope(n)

		_, err = tc.inferNodeTypes(n.Body)
		if err != nil {
			return nil, err
		}

		inferredType, err := tc.inferFunctionReturnType(n)
		if err != nil {
			return nil, err
		}
		tc.storeInferredFunctionReturnType(&n, inferredType)

		tc.popScope()

		return inferredType, nil
	case ast.ExpressionNode:
		inferredType, err := tc.inferExpressionType(n)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(node, inferredType)
		return inferredType, nil
	case ast.EnsureNode:
		_, err := tc.inferEnsureType(n)
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

		return nil, nil
	case ast.AssignmentNode:
		if err := tc.inferAssignmentTypes(n); err != nil {
			return nil, err
		}
		return nil, nil
	case ast.ReturnNode:
		return nil, nil
	}

	panic(typecheckErrorMessageWithNode(&node, fmt.Sprintf("unsupported node type %T", node)))
}

func (tc *TypeChecker) inferExpressionType(expr ast.Node) ([]ast.TypeNode, error) {
	switch e := expr.(type) {
	case ast.BinaryExpressionNode:
		inferredType, err := tc.unifyTypes(e.Left, e.Right, e.Operator)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(e, []ast.TypeNode{inferredType})
		return []ast.TypeNode{inferredType}, nil

	case ast.UnaryExpressionNode:
		inferredType, err := tc.unifyTypes(e.Operand, nil, e.Operator)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(e, []ast.TypeNode{inferredType})
		return []ast.TypeNode{inferredType}, nil

	case ast.IntLiteralNode:
		typ := ast.TypeNode{Name: ast.TypeInt}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.FloatLiteralNode:
		typ := ast.TypeNode{Name: ast.TypeFloat}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.StringLiteralNode:
		typ := ast.TypeNode{Name: ast.TypeString}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.BoolLiteralNode:
		typ := ast.TypeNode{Name: ast.TypeBool}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.VariableNode:
		// Look up the variable's type and store it for this node
		typ, err := tc.LookupVariableType(&e, tc.currentScope)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.FunctionCallNode:
		if signature, exists := tc.Functions[e.Function.Id]; exists {
			tc.storeInferredType(e, signature.ReturnTypes)
			return signature.ReturnTypes, nil
		}
		// TODO: Support built-in functions and other functions that are not defined in the current file
		// return nil, fmt.Errorf("undefined function: %s", e.Function)
		return nil, nil

	default:
		return nil, fmt.Errorf("cannot infer type for expression: %T", expr)
	}
}

// unifyTypes attempts to unify two types based on the operator and operand types
func (tc *TypeChecker) unifyTypes(left ast.Node, right ast.Node, operator ast.TokenType) (ast.TypeNode, error) {
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
	tc.storeSymbol(variable.Ident.Id, []ast.TypeNode{typ}, SymbolVariable)
	tc.storeInferredType(variable, []ast.TypeNode{typ})
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
		Ident:       fn.Ident,
		Parameters:  params,
		ReturnTypes: fn.ReturnTypes,
	}

	// Store function symbol
	tc.storeSymbol(fn.Ident.Id, fn.ReturnTypes, SymbolFunction)

	// Store parameter symbols
	for _, param := range fn.Params {
		tc.storeSymbol(param.Ident.Id, []ast.TypeNode{param.Type}, SymbolParameter)
	}
}

func (tc *TypeChecker) inferAssignmentTypes(assign ast.AssignmentNode) error {
	var resolvedTypes []ast.TypeNode

	// Collect all resolved types from RValues
	for _, value := range assign.RValues {
		if callExpr, isCall := value.(ast.FunctionCallNode); isCall {
			// Get function signature and check return types
			if sig, exists := tc.Functions[callExpr.Function.Id]; exists {
				resolvedTypes = append(resolvedTypes, sig.ReturnTypes...)
			} else {
				return fmt.Errorf("undefined function: %s", callExpr.Function.Id)
			}
		} else {
			inferredType, err := tc.inferExpressionType(value)
			if err != nil {
				return err
			}
			resolvedTypes = append(resolvedTypes, inferredType...)
		}
	}

	fmt.Println("resolvedTypes", resolvedTypes)

	// Check if number of LValues matches total resolved types
	if len(assign.LValues) != len(resolvedTypes) {
		return fmt.Errorf("assignment mismatch: %d variables but got %d values",
			len(assign.LValues), len(resolvedTypes))
	}

	// Store types for each LValue
	for i, variableNode := range assign.LValues {
		tc.storeInferredVariableType(variableNode, resolvedTypes[i])
		tc.storeInferredType(variableNode, []ast.TypeNode{resolvedTypes[i]})
	}

	return nil
}
