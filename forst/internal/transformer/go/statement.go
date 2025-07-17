package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"
	"strconv"
)

// transformErrorExpression returns an expression that represents an error when evaluated
func (t *Transformer) transformErrorExpression(stmt ast.EnsureNode) goast.Expr {
	if stmt.Error != nil {
		if errVar, ok := (*stmt.Error).(ast.EnsureErrorVar); ok {
			return &goast.Ident{
				Name: string(errVar),
			}
		}

		t.Output.EnsureImport("errors")

		return &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   goast.NewIdent("errors"),
				Sel: goast.NewIdent("New"),
			},
			Args: []goast.Expr{
				&goast.BasicLit{
					Kind:  token.STRING,
					Value: strconv.Quote((*stmt.Error).String()),
				},
			},
		}
	}

	// Get the proper base type for the assertion from inferred types
	assertionString := t.getAssertionStringForError(&stmt.Assertion)

	errorMessage := &goast.BinaryExpr{
		X: &goast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote("assertion failed: "),
		},
		Op: token.ADD,
		Y: &goast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(assertionString),
		},
	}

	t.Output.EnsureImport("errors")

	return &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   goast.NewIdent("errors"),
			Sel: goast.NewIdent("New"),
		},
		Args: []goast.Expr{
			errorMessage,
		},
	}
}

// getAssertionStringForError returns a properly qualified assertion string for error messages
func (t *Transformer) getAssertionStringForError(assertion *ast.AssertionNode) string {
	// If BaseType is set, use it with the constraint name
	if assertion.BaseType != nil {
		return assertion.ToString(assertion.BaseType)
	}

	// Otherwise, try to get the inferred type from the typechecker
	if inferredType, err := t.TypeChecker.LookupInferredType(assertion, false); err == nil && len(inferredType) > 0 {
		// Use the most specific non-hash-based alias for error messages
		nonHash := t.TypeChecker.GetMostSpecificNonHashAlias(inferredType[0])
		return assertion.ToString(&nonHash.Ident)
	}

	// Fallback to the original string representation
	return assertion.String()
}

// getZeroValue returns the zero value for a Go type
func getZeroValue(goType goast.Expr) goast.Expr {
	switch t := goType.(type) {
	case *goast.Ident:
		switch t.Name {
		case "string":
			return &goast.BasicLit{Kind: token.STRING, Value: "\"\""}
		case "int", "int64", "int32", "int16", "int8", "uint", "uint64", "uint32", "uint16", "uint8", "uintptr", "float64", "float32", "complex64", "complex128", "rune", "byte":
			return &goast.BasicLit{Kind: token.INT, Value: "0"}
		case "bool":
			return goast.NewIdent("false")
		case "error":
			return goast.NewIdent("nil")
		default:
			// For structs and user types, use T{}
			return &goast.CompositeLit{Type: t}
		}
	case *goast.StarExpr:
		return goast.NewIdent("nil")
	case *goast.ArrayType:
		return goast.NewIdent("nil")
	default:
		return goast.NewIdent("nil")
	}
}

// findBestNamedTypeForReturnType tries to find a named type that matches the given hash-based type
func (t *Transformer) findBestNamedTypeForReturnType(hashType ast.TypeNode) string {
	// If it's already a named type (not hash-based), return it
	if !hashType.IsHashBased() {
		return string(hashType.Ident)
	}

	// Look through all named types to find one that's compatible
	for typeIdent, def := range t.TypeChecker.Defs {
		if _, ok := def.(ast.TypeDefNode); ok {
			// Skip hash-based types
			if string(typeIdent)[:2] == "T_" {
				continue
			}

			// Check if this named type is compatible with the hash-based type
			if t.TypeChecker.IsTypeCompatible(hashType, ast.TypeNode{Ident: typeIdent}) {
				return string(typeIdent)
			}
		}
	}

	return ""
}

func (t *Transformer) transformErrorStatement(stmt ast.EnsureNode) goast.Stmt {
	errorExpr := t.transformErrorExpression(stmt)

	if t.isMainFunction() {
		return &goast.ExprStmt{
			X: &goast.CallExpr{
				Fun: goast.NewIdent("panic"),
				Args: []goast.Expr{
					errorExpr,
				},
			},
		}
	}

	// Look up the function's return types
	returnTypes := []goast.Expr{}
	fnNode, err := t.closestFunction()
	if err == nil {
		if fn, ok := fnNode.(ast.FunctionNode); ok {
			// Use the type checker to get the Go return types
			goReturnTypes, err := t.TypeChecker.LookupFunctionReturnType(&fn)
			if err == nil && len(goReturnTypes) > 0 {
				for i := 0; i < len(goReturnTypes)-1; i++ {
					returnType := goReturnTypes[i]
					aliasName, _ := t.TypeChecker.GetAliasedTypeName(returnType)
					if aliasName != "" && aliasName != string(returnType.Ident) {
						// If aliasName is a Go built-in type, use the correct zero value
						switch aliasName {
						case "string":
							returnTypes = append(returnTypes, &goast.BasicLit{Kind: token.STRING, Value: "\"\""})
						case "int", "int64", "int32", "int16", "int8", "uint", "uint64", "uint32", "uint16", "uint8", "uintptr", "float64", "float32", "complex64", "complex128", "rune", "byte":
							returnTypes = append(returnTypes, &goast.BasicLit{Kind: token.INT, Value: "0"})
						case "bool":
							returnTypes = append(returnTypes, goast.NewIdent("false"))
						case "error":
							returnTypes = append(returnTypes, goast.NewIdent("nil"))
						default:
							// For user-defined types, use composite literal
							returnTypes = append(returnTypes, &goast.CompositeLit{Type: goast.NewIdent(aliasName)})
						}
					} else {
						// Fallback to hash-based type
						goType, _ := t.transformType(returnType)
						returnTypes = append(returnTypes, getZeroValue(goType))
					}
				}
				// Last value is the error
				returnTypes = append(returnTypes, errorExpr)
				return &goast.ReturnStmt{Results: returnTypes}
			}
		}
	}
	// Fallback: just return the error
	return &goast.ReturnStmt{
		Results: []goast.Expr{
			errorExpr,
		},
	}
}

// transformStatement converts a Forst statement to a Go statement
func (t *Transformer) transformStatement(stmt ast.Node) (goast.Stmt, error) {
	switch s := stmt.(type) {
	case ast.EnsureNode:
		if err := t.restoreScope(stmt); err != nil {
			return nil, fmt.Errorf("failed to restore ensure statement scope: %s", err)
		}

		// Convert ensure to if statement with panic
		stmts, err := t.transformEnsureCondition(&s)
		if err != nil {
			return nil, err
		}
		var condition goast.Expr
		if len(stmts) > 0 {
			if exprStmt, ok := stmts[0].(*goast.ExprStmt); ok {
				condition = exprStmt.X
			} else {
				return nil, fmt.Errorf("expected ExprStmt from transformEnsureCondition, got %T", stmts[0])
			}
		} else {
			return nil, fmt.Errorf("transformEnsureCondition returned no statements")
		}

		// Negate for variable assertions and type guards, but not for other constraints
		finalCondition := condition
		shouldNegate := false

		// Case 1: assertion is just a variable (no constraints)
		if len(s.Assertion.Constraints) == 0 {
			shouldNegate = true
		}

		// Case 2: assertion is a type guard
		for _, constraint := range s.Assertion.Constraints {
			if t.TypeChecker.IsTypeGuardConstraint(constraint.Name) {
				shouldNegate = true
				break
			}
		}

		if shouldNegate {
			finalCondition = &goast.UnaryExpr{
				Op: token.NOT,
				X:  condition,
			}
		}

		finallyStmts := []goast.Stmt{}

		if s.Block != nil {
			if err := t.restoreScope(s.Block); err != nil {
				return nil, fmt.Errorf("failed to restore ensure statement block scope: %s", err)
			}

			for _, stmt := range s.Block.Body {
				goStmt, err := t.transformStatement(stmt)
				if err != nil {
					return nil, err
				}
				finallyStmts = append(finallyStmts, goStmt)
			}
		}

		if err := t.restoreScope(s); err != nil {
			return nil, fmt.Errorf("failed to restore ensure statement scope: %s", err)
		}

		errorStmt := t.transformErrorStatement(s)

		return &goast.IfStmt{
			Cond: finalCondition,
			Body: &goast.BlockStmt{
				List: append(finallyStmts, errorStmt),
			},
		}, nil
	case ast.ReturnNode:
		// Get the expected return types from the current function
		var expectedReturnTypes []ast.TypeNode
		if fnNode, err := t.closestFunction(); err == nil {
			if fn, ok := fnNode.(ast.FunctionNode); ok {
				rawReturnTypes, _ := t.TypeChecker.LookupFunctionReturnType(&fn)
				// Resolve aliased types from the function signature
				for _, rawType := range rawReturnTypes {
					aliasedName, err := t.TypeChecker.GetAliasedTypeName(rawType)
					if err == nil {
						aliasedType := ast.TypeNode{
							Ident:      ast.TypeIdent(aliasedName),
							TypeKind:   rawType.TypeKind,
							Assertion:  rawType.Assertion,
							TypeParams: rawType.TypeParams,
						}
						expectedReturnTypes = append(expectedReturnTypes, aliasedType)
					} else {
						expectedReturnTypes = append(expectedReturnTypes, rawType)
					}
				}
			}
		}

		results := make([]goast.Expr, len(s.Values))
		for i, value := range s.Values {
			var valueExpr goast.Expr
			var err error

			if i < len(expectedReturnTypes) {
				expectedType := &expectedReturnTypes[i]
				if shapeValue, ok := value.(ast.ShapeNode); ok {
					// Use the unified helper to determine the expected type
					context := &ShapeContext{
						ExpectedType: expectedType,
						ReturnIndex:  i,
					}
					// Try to get function name from current scope
					if fnNode, err := t.closestFunction(); err == nil {
						if fn, ok := fnNode.(ast.FunctionNode); ok {
							context.FunctionName = string(fn.Ident.ID)
						}
					}
					expectedTypeForShape := t.getExpectedTypeForShape(&shapeValue, context)
					valueExpr, err = t.transformShapeNodeWithExpectedType(&shapeValue, expectedTypeForShape)
					if err != nil {
						return nil, err
					}
				} else {
					valueExpr, err = t.transformExpression(value)
					if err != nil {
						return nil, err
					}
				}
			} else {
				valueExpr, err = t.transformExpression(value)
				if err != nil {
					return nil, err
				}
			}
			results[i] = valueExpr
		}

		// Check if we need to add missing error return
		if len(expectedReturnTypes) > len(results) {
			// Function expects more return values than provided
			// Add nil for missing error returns
			for i := len(results); i < len(expectedReturnTypes); i++ {
				expectedType := expectedReturnTypes[i]
				if expectedType.IsError() {
					results = append(results, goast.NewIdent("nil"))
				} else {
					// For non-error types, add zero value
					goType, _ := t.transformType(expectedType)
					results = append(results, getZeroValue(goType))
				}
			}
		}

		return &goast.ReturnStmt{
			Results: results,
		}, nil
	case ast.FunctionCallNode:
		// Look up parameter types for the function
		paramTypes := make([]ast.TypeNode, len(s.Arguments))
		if sig, ok := t.TypeChecker.Functions[s.Function.ID]; ok && len(sig.Parameters) == len(s.Arguments) {
			for i, param := range sig.Parameters {
				if param.Type.Ident == ast.TypeAssertion && param.Type.Assertion != nil {
					inferredTypes, err := t.TypeChecker.InferAssertionType(param.Type.Assertion, false, "", nil)
					if err == nil && len(inferredTypes) > 0 {
						paramTypes[i] = inferredTypes[0]
					} else {
						paramTypes[i] = param.Type
					}
				} else {
					paramTypes[i] = param.Type
				}
			}
		}
		args := make([]goast.Expr, len(s.Arguments))
		for i, arg := range s.Arguments {
			if shapeArg, ok := arg.(ast.ShapeNode); ok && paramTypes[i].Ident != ast.TypeImplicit {
				// Use the unified helper to determine the expected type
				context := &ShapeContext{
					ExpectedType:   &paramTypes[i],
					FunctionName:   string(s.Function.ID),
					ParameterIndex: i,
				}
				expectedTypeForShape := t.getExpectedTypeForShape(&shapeArg, context)
				argExpr, err := t.transformShapeNodeWithExpectedType(&shapeArg, expectedTypeForShape)
				if err != nil {
					return nil, err
				}
				args[i] = argExpr
			} else {
				argExpr, err := t.transformExpression(arg)
				if err != nil {
					return nil, err
				}
				args[i] = argExpr
			}
		}
		return &goast.ExprStmt{
			X: &goast.CallExpr{
				Fun:  goast.NewIdent(s.Function.String()),
				Args: args,
			},
		}, nil
	case ast.AssignmentNode:
		// Check for explicit type annotation
		if len(s.ExplicitTypes) > 0 && s.ExplicitTypes[0] != nil {
			varName := s.LValues[0].Ident.String()

			var typeExpr goast.Expr
			var expectedType *ast.TypeNode
			if t != nil {
				typeIdent, err := t.transformType(*s.ExplicitTypes[0])
				if err != nil {
					typeExpr = goast.NewIdent(string(s.ExplicitTypes[0].Ident))
					expectedType = s.ExplicitTypes[0]
				} else {
					typeExpr = typeIdent
					expectedType = s.ExplicitTypes[0]
				}
			} else {
				typeExpr = goast.NewIdent(string(s.ExplicitTypes[0].Ident))
				expectedType = s.ExplicitTypes[0]
			}

			if shapeRHS, ok := s.RValues[0].(ast.ShapeNode); ok {
				// Use the unified helper to determine the expected type
				context := &ShapeContext{
					ExpectedType: expectedType,
					VariableName: varName,
				}
				expectedTypeForShape := t.getExpectedTypeForShape(&shapeRHS, context)
				rhs, err := t.transformShapeNodeWithExpectedType(&shapeRHS, expectedTypeForShape)
				if err != nil {
					return nil, err
				}
				return &goast.DeclStmt{
					Decl: &goast.GenDecl{
						Tok: token.VAR,
						Specs: []goast.Spec{
							&goast.ValueSpec{
								Names:  []*goast.Ident{goast.NewIdent(varName)},
								Type:   typeExpr,
								Values: []goast.Expr{rhs},
							},
						},
					},
				}, nil
			}
			rhs, err := t.transformExpression(s.RValues[0])
			if err != nil {
				return nil, err
			}
			return &goast.DeclStmt{
				Decl: &goast.GenDecl{
					Tok: token.VAR,
					Specs: []goast.Spec{
						&goast.ValueSpec{
							Names:  []*goast.Ident{goast.NewIdent(varName)},
							Type:   typeExpr,
							Values: []goast.Expr{rhs},
						},
					},
				},
			}, nil
		}
		lhs := make([]goast.Expr, len(s.LValues))
		for i, lval := range s.LValues {
			lhsExpr, err := t.transformExpression(lval)
			if err != nil {
				return nil, err
			}
			lhs[i] = lhsExpr
		}
		rhs := make([]goast.Expr, len(s.RValues))
		for i, rval := range s.RValues {
			if shapeRHS, ok := rval.(ast.ShapeNode); ok && len(s.LValues) == 1 {
				varName := s.LValues[0].Ident.String()
				rhsExpr, err := t.transformShapeNodeWithExpectedType(&shapeRHS, t.getExpectedTypeForShape(&shapeRHS, &ShapeContext{VariableName: varName}))
				if err != nil {
					return nil, err
				}
				rhs[i] = rhsExpr
			} else {
				rhsExpr, err := t.transformExpression(rval)
				if err != nil {
					return nil, err
				}
				rhs[i] = rhsExpr
			}
		}
		operator := token.ASSIGN
		if s.IsShort {
			operator = token.DEFINE
		}
		return &goast.AssignStmt{
			Lhs: lhs,
			Tok: operator,
			Rhs: rhs,
		}, nil
	default:
		return &goast.EmptyStmt{}, nil
	}
}
