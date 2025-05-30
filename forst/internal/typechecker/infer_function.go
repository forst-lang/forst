package typechecker

import (
	"fmt"
	"forst/internal/ast"
)

// Infers the return type of a function from its body
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
