package typechecker

import (
	"fmt"
	"forst/pkg/ast"

	log "github.com/sirupsen/logrus"
)

// inferFunctionReturnType infers the return type of a function from its body
func (tc *TypeChecker) inferFunctionReturnType(fn ast.FunctionNode) ([]ast.TypeNode, error) {
	parsedType := fn.ReturnTypes
	inferredType := []ast.TypeNode{}

	// For empty functions, default to void return type
	if len(fn.Body) == 0 {
		return ensureMatching(fn, inferredType, parsedType, "Empty function is not void")
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
				if retType.Ident != firstType[0].Ident {
					return nil, failWithTypeMismatch(fn, inferredType, firstType, "Inconsistent type of return statements")
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
				if exprType.Ident != returnStmtTypes[0][i].Ident {
					return nil, failWithTypeMismatch(fn, inferredType, exprTypes, "Inconsistent return expression type")
				}
			}
		}

		return ensureMatching(fn, inferredType, parsedType, "Invalid return expression type")
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
					{Ident: ast.TypeError},
				}
			} else {
				if len(inferredType) != 1 && len(inferredType) != 2 {
					return nil, fmt.Errorf("ensure statements require the function to return an error or a tuple with an error as the second type, got %s", formatTypeList(inferredType))
				}

				// TODO: If parsed types are empty and inferred type is a single (non-error) return type, just append the error type to the inferred return type
				if inferredType[len(inferredType)-1].Ident != ast.TypeError {
					return nil, fmt.Errorf("ensure statements require the function to an error as the second return type, got %s", inferredType[1].Ident)
				}
			}
		}
	}

	if len(inferredType) == 0 {
		inferredType = []ast.TypeNode{{Ident: ast.TypeVoid}}
	}

	return ensureMatching(fn, inferredType, parsedType, "Invalid return type")
}

// TODO: Improve type inference for complex types
// This should handle:
// 1. Binary type expressions
// 2. Nested shapes
// 3. Type aliases
// 4. Generic types
func (tc *TypeChecker) inferShapeType(shape *ast.ShapeNode) ([]ast.TypeNode, error) {
	hash := tc.Hasher.HashNode(shape)
	typeIdent := hash.ToTypeIdent()
	shapeType := []ast.TypeNode{
		{
			Ident: typeIdent,
		},
	}
	for name, field := range shape.Fields {
		if field.Shape != nil {
			fieldType, err := tc.inferShapeType(field.Shape)
			if err != nil {
				return nil, err
			}

			fieldHash := tc.Hasher.HashNode(field)
			fieldTypeIdent := fieldHash.ToTypeIdent()
			log.Tracef("Inferred type of shape field %s: %s, field: %s", name, fieldTypeIdent, field)
			tc.storeInferredType(field.Shape, fieldType)
		} else if field.Assertion != nil {
			// Skip if the assertion type has already been inferred
			inferredType, _ := tc.inferAssertionType(field.Assertion, false)
			if inferredType != nil {
				continue
			}

			fieldHash := tc.Hasher.HashNode(field)
			fieldTypeIdent := fieldHash.ToTypeIdent()
			log.Tracef("Inferred type of assertion field %s: %s", name, fieldTypeIdent)
			tc.registerType(ast.TypeDefNode{
				Ident: fieldTypeIdent,
				Expr: ast.TypeDefAssertionExpr{
					Assertion: field.Assertion,
				},
			})
		} else {
			panic(fmt.Sprintf("Shape field has neither assertion nor shape: %T", field))
		}
	}

	tc.storeInferredType(shape, shapeType)
	log.Tracef("Inferred shape type: %s", shapeType)

	// The type is not registered here as the specific type implementation
	// depends on the target language and will be determined in the transformer.

	return shapeType, nil
}

// TODO: Improve assertion type inference
// This should handle:
// 1. Complex constraints
// 2. Nested assertions
// 3. Type aliases
// 4. Generic types
func (tc *TypeChecker) inferAssertionType(assertion *ast.AssertionNode, requireInferred bool) ([]ast.TypeNode, error) {
	// Check if we've already inferred this assertion's type
	existingTypes, err := tc.LookupInferredType(assertion, requireInferred)
	if err != nil {
		return nil, err
	}

	if existingTypes != nil {
		return existingTypes, nil
	}

	hash := tc.Hasher.HashNode(assertion)
	typeIdent := hash.ToTypeIdent()
	typeNode := ast.TypeNode{
		Ident:     typeIdent,
		Assertion: assertion,
	}
	inferredType := []ast.TypeNode{typeNode}
	tc.storeInferredType(assertion, inferredType)

	// Process each constraint in the assertion
	tc.registerType(ast.TypeDefNode{
		Ident: typeIdent,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: assertion,
		},
	})

	if assertion.BaseType != nil && (*assertion.BaseType == "trpc.Mutation" || *assertion.BaseType == "trpc.Query") {
		for _, constraint := range assertion.Constraints {
			switch constraint.Name {
			case "Input":
				if len(constraint.Args) != 1 {
					return nil, fmt.Errorf("input constraint must have exactly one argument")
				}
				arg := constraint.Args[0]
				if arg.Shape != nil {
					if _, err := tc.inferShapeType(arg.Shape); err != nil {
						return nil, fmt.Errorf("failed to infer shape type: %w", err)
					}
				}
			}
		}
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
	log.Tracef("inferNodeType: %s", node.String())
	// Check if we've already inferred this node's type
	alreadyInferredType, err := tc.LookupInferredType(node, false)
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

		// Convert []ParamNode to []Node
		params := make([]ast.Node, len(n.Params))
		for i, param := range n.Params {
			params[i] = param
		}

		_, err = tc.inferNodeTypes(params)
		if err != nil {
			return nil, err
		}

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
	case ast.SimpleParamNode:
		if n.Type.Assertion != nil {
			inferredType, err := tc.inferAssertionType(n.Type.Assertion, false)
			if err != nil {
				return nil, err
			}
			return inferredType, nil
		}
		return []ast.TypeNode{n.Type}, nil
	case ast.DestructuredParamNode:
		return nil, nil
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

	case ast.AssertionNode:
		_, err := tc.inferAssertionType(&n, false)
		if err != nil {
			return nil, err
		}
		return nil, nil

	case ast.TypeNode:
		return nil, nil

	case ast.TypeDefNode:
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
		typ := ast.TypeNode{Ident: ast.TypeInt}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.FloatLiteralNode:
		typ := ast.TypeNode{Ident: ast.TypeFloat}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.StringLiteralNode:
		typ := ast.TypeNode{Ident: ast.TypeString}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.BoolLiteralNode:
		typ := ast.TypeNode{Ident: ast.TypeBool}
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
