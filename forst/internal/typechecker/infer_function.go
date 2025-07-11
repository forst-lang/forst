package typechecker

import (
	"fmt"
	"forst/internal/ast"
)

// Infers the return type of a function from its body
// This function should be called while the function scope is active
func (tc *TypeChecker) inferFunctionReturnType(fn ast.FunctionNode) ([]ast.TypeNode, error) {
	parsedType := fn.ReturnTypes
	inferredType := []ast.TypeNode{}

	// For empty functions, default to void return type
	if len(fn.Body) == 0 {
		return ensureMatching(fn, inferredType, parsedType, "Empty function is not void")
	}

	// Check if there are any ensure nodes and determine if function should return error
	hasEnsure := false
	for _, stmt := range fn.Body {
		if _, ok := stmt.(ast.EnsureNode); ok {
			hasEnsure = true
			break
		}
	}

	// Find all return statements and collect their types
	returnStmtTypes := make([][]ast.TypeNode, 0)
	for _, stmt := range fn.Body {
		if retStmt, ok := stmt.(ast.ReturnNode); ok {
			// Get types of all return values
			retTypes := make([]ast.TypeNode, 0)
			for i, value := range retStmt.Values {
				// Contextual typing for nil
				if value.Kind() == ast.NodeKindNilLiteral {
					// Try to get expected type from parsedType, function signature, or previous returns
					var expectedType ast.TypeNode
					if len(parsedType) > i {
						expectedType = parsedType[i]
					} else if len(fn.ReturnTypes) > i {
						expectedType = fn.ReturnTypes[i]
					} else if len(returnStmtTypes) > 0 && len(returnStmtTypes[0]) > i {
						expectedType = returnStmtTypes[0][i]
					} else if hasEnsure && i == 1 {
						// If there's an ensure statement and this is the second return value, expect Error
						expectedType = ast.TypeNode{Ident: ast.TypeError}
					}
					if isNilableType(tc, expectedType) {
						retTypes = append(retTypes, expectedType)
					} else {
						return nil, fmt.Errorf("'nil' used as return value but expected type is not nilable (got %s)", expectedType.Ident)
					}
				} else {
					retType, err := tc.inferExpressionType(value)
					if err != nil {
						return nil, err
					}
					if len(retType) != 1 {
						return nil, fmt.Errorf("return value expression must return exactly one type, got %d", len(retType))
					}
					retTypes = append(retTypes, retType[0])
				}
			}
			returnStmtTypes = append(returnStmtTypes, retTypes)
		}
	}

	// If we found return statements, verify they all have the same type
	if len(returnStmtTypes) > 1 {
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

	// Handle ensure statements
	if hasEnsure {
		if len(inferredType) == 0 {
			inferredType = []ast.TypeNode{
				{Ident: ast.TypeError},
			}
		} else {
			if len(inferredType) < 1 || len(inferredType) > 2 {
				return nil, fmt.Errorf("ensure statements require the function to return an error or a tuple with an error as the second type, got %s", formatTypeList(inferredType))
			}

			// If the inferred type is a single (non-error) return type, just append the error type to the inferred return type
			if len(inferredType) == 1 && inferredType[0].Ident != ast.TypeError {
				inferredType = append(inferredType, ast.TypeNode{Ident: ast.TypeError})
			}

			// If parsed types are empty and inferred type is a single (non-error) return type, just append the error type to the inferred return type
			if inferredType[len(inferredType)-1].Ident != ast.TypeError {
				// Special case: if the last return type is nil, it's not a valid return type
				// and we should force the function to return an error
				inferredType[len(inferredType)-1] = ast.TypeNode{Ident: ast.TypeError}
			}
		}
	}

	if len(inferredType) == 0 {
		inferredType = []ast.TypeNode{{Ident: ast.TypeVoid}}
	}

	return ensureMatching(fn, inferredType, parsedType, "Invalid return type")
}

// Helper: isNilableType checks if a type can be assigned nil
func isNilableType(tc *TypeChecker, t ast.TypeNode) bool {
	// Follow type aliases to the base type
	base := t
	chain := tc.GetTypeAliasChain(t)
	if len(chain) > 0 {
		base = chain[len(chain)-1]
	}

	// Check if the base type is nilable
	switch base.Ident {
	case ast.TypePointer, ast.TypeError, ast.TypeMap, ast.TypeArray:
		return true
	}

	// Also check string versions for built-in types
	switch string(base.Ident) {
	case "Pointer", "Error", "Map", "Array":
		return true
	}

	return false
}
