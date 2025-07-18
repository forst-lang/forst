package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"
	"strconv"

	logrus "github.com/sirupsen/logrus"
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

// findBestNamedTypeForReturnStructLiteral finds the best named type for a struct literal in a return statement.
// It always attempts to find a structurally identical named type, even if the expected type is not hash-based.
func (t *Transformer) findBestNamedTypeForReturnStructLiteral(shapeNode ast.ShapeNode, expectedType *ast.TypeNode) *ast.TypeNode {
	// Always try to find a structurally identical named type for the shape
	if namedType := t.TypeChecker.FindAnyStructurallyIdenticalNamedType(shapeNode); namedType != "" {
		return &ast.TypeNode{Ident: namedType}
	}

	// If no structurally identical named type found, use the expected type
	if expectedType != nil {
		return expectedType
	}

	return nil
}

func (t *Transformer) transformErrorStatement(stmt ast.EnsureNode) goast.Stmt {
	// DEBUG: Log entry to error statement with detailed type information
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function": "transformErrorStatement",
			"stmt":     stmt.String(),
		}).Error("[PINPOINT] transformErrorStatement called")
	}

	errorExpr := t.transformErrorExpression(stmt)

	// Get the function's declared return type from the current scope
	var goReturnTypes []ast.TypeNode
	var functionName string
	fnNode, err := t.closestFunction()
	if err == nil {
		if fn, ok := fnNode.(ast.FunctionNode); ok {
			functionName = string(fn.Ident.ID)
			if sig, exists := t.TypeChecker.Functions[fn.Ident.ID]; exists {
				goReturnTypes = sig.ReturnTypes

				// DEBUG: Log the function signature and return types
				if t.log != nil {
					t.log.WithFields(logrus.Fields{
						"function":     "transformErrorStatement",
						"functionName": functionName,
						"returnTypes":  fmt.Sprintf("%v", goReturnTypes),
						"returnCount":  len(goReturnTypes),
					}).Warn("[PINPOINT] Function signature for error return")
				}
			}
		}
	}

	// If we can't find the function signature, fallback to a single error return
	if len(goReturnTypes) == 0 {
		return &goast.ReturnStmt{
			Results: []goast.Expr{errorExpr},
		}
	}

	// Always build a zero value for the function's declared return type(s)
	zeroValues := make([]goast.Expr, 0, len(goReturnTypes))
	for i, retType := range goReturnTypes {
		// DEBUG: Log each return type being processed
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function":     "transformErrorStatement",
				"functionName": functionName,
				"returnIndex":  i,
				"returnType":   retType.Ident,
				"typeKind":     retType.TypeKind,
			}).Warn("[PINPOINT] Processing return type for zero value")
		}

		zeroValue := t.buildZeroCompositeLiteral(&retType)
		zeroValues = append(zeroValues, zeroValue)
	}

	// Replace the last value with the error expression (assuming last return is error)
	if len(zeroValues) > 0 {
		zeroValues[len(zeroValues)-1] = errorExpr
	}

	return &goast.ReturnStmt{
		Results: zeroValues,
	}
}

// Helper: Build a composite literal of the expected type, mapping a variable to the first field, zero for others
func (t *Transformer) buildCompositeLiteralForReturn(expectedType *ast.TypeNode, valueVar string) goast.Expr {
	def, ok := t.TypeChecker.Defs[expectedType.Ident].(ast.TypeDefNode)
	if !ok {
		return goast.NewIdent(valueVar)
	}
	shapeExpr, ok := def.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		return goast.NewIdent(valueVar)
	}
	fields := shapeExpr.Shape.Fields
	elts := make([]goast.Expr, 0, len(fields))
	used := false
	for fieldName, field := range fields {
		var val goast.Expr
		if !used {
			val = goast.NewIdent(valueVar)
			used = true
		} else {
			goFieldType, _ := t.transformType(*field.Type)
			val = getZeroValue(goFieldType)
		}
		elts = append(elts, &goast.KeyValueExpr{
			Key:   goast.NewIdent(fieldName),
			Value: val,
		})
	}
	return &goast.CompositeLit{Type: goast.NewIdent(string(expectedType.Ident)), Elts: elts}
}

// Helper: Build a composite literal of the expected type, all zero values
func (t *Transformer) buildZeroCompositeLiteral(expectedType *ast.TypeNode) goast.Expr {
	// DEBUG: Log the expected type
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function":     "buildZeroCompositeLiteral",
			"expectedType": string(expectedType.Ident),
			"typeKind":     expectedType.TypeKind,
			"isHashBased":  expectedType.IsHashBased(),
		}).Warn("[DEBUG] buildZeroCompositeLiteral called")
	}

	def, ok := t.TypeChecker.Defs[expectedType.Ident].(ast.TypeDefNode)
	if !ok {
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function":     "buildZeroCompositeLiteral",
				"expectedType": string(expectedType.Ident),
				"found":        false,
			}).Warn("[DEBUG] Type definition not found")
		}
		return &goast.CompositeLit{Type: goast.NewIdent(string(expectedType.Ident))}
	}
	shapeExpr, ok := def.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function":     "buildZeroCompositeLiteral",
				"expectedType": string(expectedType.Ident),
				"exprType":     fmt.Sprintf("%T", def.Expr),
			}).Warn("[DEBUG] Type definition is not a shape expression")
		}
		return &goast.CompositeLit{Type: goast.NewIdent(string(expectedType.Ident))}
	}
	fields := shapeExpr.Shape.Fields
	elts := make([]goast.Expr, 0, len(fields))
	for fieldName, field := range fields {
		goFieldType, _ := t.transformType(*field.Type)
		val := getZeroValue(goFieldType)
		elts = append(elts, &goast.KeyValueExpr{
			Key:   goast.NewIdent(fieldName),
			Value: val,
		})
	}
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function":      "buildZeroCompositeLiteral",
			"expectedType":  string(expectedType.Ident),
			"fieldCount":    len(fields),
			"generatedType": string(expectedType.Ident),
		}).Warn("[DEBUG] Generated zero composite literal")
	}
	return &goast.CompositeLit{Type: goast.NewIdent(string(expectedType.Ident)), Elts: elts}
}

// transformStatement converts a Forst statement to a Go statement
func (t *Transformer) transformStatement(stmt ast.Node) (goast.Stmt, error) {
	// DEBUG: Log entry to transformStatement with detailed type information
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function": "transformStatement",
			"stmtType": fmt.Sprintf("%T", stmt),
			"stmt":     stmt.String(),
		}).Error("[PINPOINT] transformStatement called")
	}

	switch s := stmt.(type) {
	case ast.EnsureNode:
		// DEBUG: Log when EnsureNode is processed
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function": "transformStatement",
				"stmtType": "EnsureNode",
				"stmt":     s.String(),
			}).Error("[PINPOINT] Processing EnsureNode")
		}
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function":           "transformStatement",
				"scopeBeforeRestore": fmt.Sprintf("%v", t.currentScope()),
				"nodeType":           fmt.Sprintf("%T", stmt),
			}).Warn("[DEBUG] Before restoreScope(EnsureNode)")
		}
		if err := t.restoreScope(stmt); err != nil {
			return nil, fmt.Errorf("failed to restore ensure statement scope: %s", err)
		}
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function":          "transformStatement",
				"scopeAfterRestore": fmt.Sprintf("%v", t.currentScope()),
				"nodeType":          fmt.Sprintf("%T", stmt),
			}).Warn("[DEBUG] After restoreScope(EnsureNode)")
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

		// Always restore the function scope before calling transformErrorStatement
		if fnNode, err := t.closestFunction(); err == nil {
			if err := t.restoreScope(fnNode); err != nil {
				return nil, fmt.Errorf("failed to restore function scope before error statement: %s", err)
			}
		}

		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function":             "transformStatement",
				"scopeBeforeErrorStmt": fmt.Sprintf("%v", t.currentScope()),
				"nodeType":             fmt.Sprintf("%T", stmt),
			}).Warn("[DEBUG] Before transformErrorStatement")
		}

		errorStmt := t.transformErrorStatement(s)

		return &goast.IfStmt{
			Cond: finalCondition,
			Body: &goast.BlockStmt{
				List: append(finallyStmts, errorStmt),
			},
		}, nil
	case ast.ReturnNode:
		// DEBUG: Log when ReturnNode is processed
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function": "transformStatement",
				"stmtType": "ReturnNode",
				"stmt":     s.String(),
			}).Error("[PINPOINT] Processing ReturnNode")
		}
		// DEBUG: Log entry
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function": "transformReturnStatement",
				"values":   len(s.Values),
			}).Warn("transformReturnStatement called")
		}

		// Find the current function node and its inferred return types
		var expectedReturnTypes []ast.TypeNode
		var functionName string
		if fnNode, err := t.closestFunction(); err == nil {
			if fn, ok := fnNode.(ast.FunctionNode); ok {
				functionName = string(fn.Ident.ID)
				// DEBUG: Log function lookup
				if t.log != nil {
					t.log.WithFields(logrus.Fields{
						"function":      "transformReturnStatement",
						"functionName":  fn.Ident.ID,
						"functionFound": true,
					}).Warn("Function node found")
				}

				// Always get the inferred return type from the typechecker
				if sig, ok := t.TypeChecker.Functions[fn.Ident.ID]; ok && len(sig.ReturnTypes) > 0 {
					expectedReturnTypes = sig.ReturnTypes

					// DEBUG: Log function signature
					if t.log != nil {
						t.log.WithFields(logrus.Fields{
							"function":     "transformReturnStatement",
							"functionName": fn.Ident.ID,
							"returnTypes":  fmt.Sprintf("%v", expectedReturnTypes),
						}).Warn("Function signature for return statement")
					}
				} else {
					// DEBUG: Log missing signature
					if t.log != nil {
						t.log.WithFields(logrus.Fields{
							"function":       "transformReturnStatement",
							"functionName":   fn.Ident.ID,
							"hasSig":         ok,
							"sigReturnTypes": len(sig.ReturnTypes),
						}).Warn("No function signature found")
					}
				}
			} else {
				// DEBUG: Log function type mismatch
				if t.log != nil {
					t.log.WithFields(logrus.Fields{
						"function": "transformReturnStatement",
						"fnType":   fmt.Sprintf("%T", fnNode),
					}).Warn("Function node is not FunctionNode")
				}
			}
		} else {
			// DEBUG: Log function lookup error
			if t.log != nil {
				t.log.WithFields(logrus.Fields{
					"function": "transformReturnStatement",
					"error":    err.Error(),
				}).Warn("Failed to find closest function")
			}
		}

		results := make([]goast.Expr, len(s.Values))
		for i, value := range s.Values {
			var expectedType *ast.TypeNode
			if i < len(expectedReturnTypes) {
				expectedType = &expectedReturnTypes[i]
			}

			// PINPOINT: Log mapping from AST node to Go type for each return value
			if t.log != nil {
				t.log.WithFields(logrus.Fields{
					"function":     functionName,
					"returnIndex":  i,
					"expectedType": expectedType,
					"actualAST":    fmt.Sprintf("%T", value),
				}).Warn("[PINPOINT] Mapping AST node to Go type for return value")
			}

			// If the expected type is a named struct and the value is a variable or shape, wrap it in the expected type
			if expectedType != nil && expectedType.IsUserDefined() {
				// PINPOINT: Log when we have a user-defined expected type
				if t.log != nil {
					t.log.WithFields(logrus.Fields{
						"function":     functionName,
						"returnIndex":  i,
						"expectedType": expectedType.Ident,
						"valueType":    fmt.Sprintf("%T", value),
					}).Warn("[PINPOINT] Found user-defined expected type for return value")
				}
				// Check if the value is a variable or shape that needs to be wrapped
				switch v := value.(type) {
				case ast.VariableNode:
					// PINPOINT: Log when wrapping variable in named struct
					if t.log != nil {
						t.log.WithFields(logrus.Fields{
							"function":     functionName,
							"returnIndex":  i,
							"variableName": v.Ident.ID,
							"expectedType": expectedType.Ident,
						}).Warn("[PINPOINT] Wrapping variable in named struct")
					}
					// If it's a variable, wrap it in the expected struct type
					expr, err := t.wrapVariableInNamedStruct(expectedType, v)
					if err != nil {
						return nil, fmt.Errorf("transformReturnStatement: %w", err)
					}
					results[i] = expr
					continue
				case ast.ShapeNode, *ast.ShapeNode:
					shapeNode, ok := getShapeNode(value)
					if ok {
						// PINPOINT: Log when wrapping shape in named struct (robust)
						if t.log != nil {
							t.log.WithFields(logrus.Fields{
								"function":     functionName,
								"returnIndex":  i,
								"shapeFields":  len(shapeNode.Fields),
								"expectedType": expectedType.Ident,
							}).Warn("[PINPOINT] Wrapping shape (robust) in named struct")
						}
						expr, err := t.transformShapeNodeWithExpectedType(shapeNode, expectedType)
						if err != nil {
							return nil, fmt.Errorf("transformReturnStatement: failed to transform shape node: %w", err)
						}
						results[i] = expr
						continue
					}
				default:
					// PINPOINT: Log when value is not a recognized type
					if t.log != nil {
						t.log.WithFields(logrus.Fields{
							"function":    functionName,
							"returnIndex": i,
							"actualType":  fmt.Sprintf("%T", value),
						}).Warn("[PINPOINT] Unhandled value type in return statement switch")
					}
				}
			}

			var valueExpr goast.Expr
			var err error

			if i < len(expectedReturnTypes) {
				expectedType := &expectedReturnTypes[i]
				if expectedType.TypeKind == ast.TypeKindUserDefined {
					for i, ret := range results {
						if ident, ok := ret.(*goast.Ident); ok {
							goRet := t.buildCompositeLiteralForReturn(expectedType, ident.Name)
							results[i] = goRet
						}
					}
				} else {
					if shapeValue, ok := value.(ast.ShapeNode); ok {
						// PINPOINT: Log when processing shape literal without user-defined type
						if t.log != nil {
							t.log.WithFields(logrus.Fields{
								"function":      functionName,
								"returnIndex":   i,
								"expectedType":  expectedType.Ident,
								"isUserDefined": expectedType.IsUserDefined(),
							}).Warn("[PINPOINT] Processing shape literal without user-defined expected type")
						}

						context := &ShapeContext{
							ExpectedType: expectedType,
							ReturnIndex:  i,
						}
						if fnNode, err := t.closestFunction(); err == nil {
							if fn, ok := fnNode.(ast.FunctionNode); ok {
								context.FunctionName = string(fn.Ident.ID)
							}
						}
						expectedTypeForShape := t.getExpectedTypeForShape(&shapeValue, context)

						// Use the new helper function to find the best named type for return struct literals
						useType := t.findBestNamedTypeForReturnStructLiteral(shapeValue, expectedTypeForShape)

						// PINPOINT: Log the type selection process for struct literal
						if t.log != nil {
							t.log.WithFields(logrus.Fields{
								"function":     "transformReturnStatement",
								"returnIndex":  i,
								"expectedType": expectedTypeForShape,
								"selectedType": useType,
								"isHashBased":  expectedTypeForShape != nil && expectedTypeForShape.IsHashBased(),
							}).Warn("[PINPOINT] Processing struct literal in return statement")
						}
						valueExpr, err = t.transformShapeNodeWithExpectedType(&shapeValue, useType)
						if err != nil {
							return nil, err
						}
					} else {
						// PINPOINT: Log when processing non-shape value
						if t.log != nil {
							t.log.WithFields(logrus.Fields{
								"function":     functionName,
								"returnIndex":  i,
								"valueType":    fmt.Sprintf("%T", value),
								"expectedType": expectedType.Ident,
							}).Warn("[PINPOINT] Processing non-shape return value")
						}
						valueExpr, err = t.transformExpression(value)
						if err != nil {
							return nil, err
						}
					}
				}
			} else {
				// PINPOINT: Log when no expected type available
				if t.log != nil {
					t.log.WithFields(logrus.Fields{
						"function":         functionName,
						"returnIndex":      i,
						"valueType":        fmt.Sprintf("%T", value),
						"expectedTypesLen": len(expectedReturnTypes),
					}).Warn("[PINPOINT] No expected type available for return value")
				}
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

		// DEBUG: Log function signature information
		if t.log != nil {
			if fnNode, err := t.closestFunction(); err == nil {
				if fn, ok := fnNode.(ast.FunctionNode); ok {
					t.log.WithFields(logrus.Fields{
						"function":     "transformReturnStatement",
						"functionName": string(fn.Ident.ID),
						"returnTypes":  expectedReturnTypes,
					}).Warn("Function signature for return statement")
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

// wrapInNamedStruct generates a Go composite literal of the expected named struct type, mapping the fields from the variable or shape node.
func (t *Transformer) wrapInNamedStruct(expectedType *ast.TypeNode, value ast.Node) (goast.Expr, error) {
	goType, _ := t.transformType(*expectedType)

	switch v := value.(type) {
	case ast.ShapeNode:
		expr, err := t.transformShapeNodeWithExpectedType(&v, expectedType)
		return expr, err
	case ast.VariableNode:
		if def, ok := t.TypeChecker.Defs[expectedType.Ident].(ast.TypeDefNode); ok {
			if shapeExpr, ok := def.Expr.(ast.TypeDefShapeExpr); ok {
				fields := shapeExpr.Shape.Fields
				if len(fields) == 1 {
					for fieldName := range fields {
						return &goast.CompositeLit{
							Type: goType,
							Elts: []goast.Expr{
								&goast.KeyValueExpr{
									Key:   goast.NewIdent(fieldName),
									Value: goast.NewIdent(string(v.Ident.ID)),
								},
							},
						}, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("wrapInNamedStruct: cannot wrap variable in struct with multiple fields or missing type definition: %v", expectedType.Ident)
	default:
		return nil, fmt.Errorf("wrapInNamedStruct: unsupported node type %T for expected type %v", value, expectedType.Ident)
	}
}

// wrapShapeInNamedStructIfNeeded checks if a shape literal should be wrapped in the expected named struct type.
func (t *Transformer) wrapShapeInNamedStructIfNeeded(expectedType *ast.TypeNode, shape ast.ShapeNode) (goast.Expr, error) {
	// DEBUG: Log entry
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function":     "wrapShapeInNamedStructIfNeeded",
			"expectedType": expectedType.Ident,
			"shapeFields":  fmt.Sprintf("%+v", shape.Fields),
		}).Warn("wrapShapeInNamedStructIfNeeded called")
	}

	// First, try to use the existing wrapInNamedStruct logic
	expr, err := t.wrapInNamedStruct(expectedType, shape)
	if err == nil {
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function": "wrapShapeInNamedStructIfNeeded",
				"result":   "used wrapInNamedStruct",
			}).Warn("wrapShapeInNamedStructIfNeeded: used wrapInNamedStruct")
		}
		return expr, nil
	}

	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function": "wrapShapeInNamedStructIfNeeded",
			"error":    err.Error(),
		}).Warn("wrapShapeInNamedStructIfNeeded: wrapInNamedStruct failed, trying field matching")
	}

	// If that fails, check if the shape should be wrapped in the expected type
	// Get the expected struct definition
	def, ok := t.TypeChecker.Defs[expectedType.Ident].(ast.TypeDefNode)
	if !ok {
		return nil, fmt.Errorf("wrapShapeInNamedStructIfNeeded: expected type %v is not a TypeDefNode", expectedType.Ident)
	}

	shapeExpr, ok := def.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		return nil, fmt.Errorf("wrapShapeInNamedStructIfNeeded: expected type %v does not have a shape expression", expectedType.Ident)
	}

	// Check if the shape fields match the expected struct fields
	expectedFields := shapeExpr.Shape.Fields
	shapeFields := shape.Fields

	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function":       "wrapShapeInNamedStructIfNeeded",
			"expectedFields": fmt.Sprintf("%+v", expectedFields),
			"shapeFields":    fmt.Sprintf("%+v", shapeFields),
			"lenExpected":    len(expectedFields),
			"lenShape":       len(shapeFields),
		}).Warn("wrapShapeInNamedStructIfNeeded: comparing field counts")
	}

	// If the shape has the same field names as the expected struct, wrap it
	if len(shapeFields) == len(expectedFields) {
		matches := true
		for fieldName := range shapeFields {
			if _, exists := expectedFields[fieldName]; !exists {
				matches = false
				break
			}
		}

		if matches {
			if t.log != nil {
				t.log.WithFields(logrus.Fields{
					"function": "wrapShapeInNamedStructIfNeeded",
					"result":   "field names match, wrapping",
				}).Warn("wrapShapeInNamedStructIfNeeded: field names match, wrapping")
			}

			// The shape matches the expected struct structure, wrap it
			goType, _ := t.transformType(*expectedType)
			elts := make([]goast.Expr, 0, len(shapeFields))

			for fieldName, field := range shapeFields {
				var fieldValue goast.Expr
				var err error

				if field.Node != nil {
					if exprNode, ok := field.Node.(ast.ExpressionNode); ok {
						fieldValue, err = t.transformExpression(exprNode)
						if err != nil {
							return nil, fmt.Errorf("transformReturnStatement: failed to transform field %s: %w", fieldName, err)
						}
					} else {
						return nil, fmt.Errorf("transformReturnStatement: field %s is not an ExpressionNode", fieldName)
					}
				} else {
					fieldValue = goast.NewIdent("nil")
				}

				if err != nil {
					return nil, fmt.Errorf("wrapShapeInNamedStructIfNeeded: failed to transform field %s: %w", fieldName, err)
				}

				elts = append(elts, &goast.KeyValueExpr{
					Key:   goast.NewIdent(fieldName),
					Value: fieldValue,
				})
			}

			return &goast.CompositeLit{
				Type: goType,
				Elts: elts,
			}, nil
		}
	}

	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function": "wrapShapeInNamedStructIfNeeded",
			"result":   "falling back to standard transformation",
		}).Warn("wrapShapeInNamedStructIfNeeded: falling back to standard transformation")
	}

	// If we get here, the shape doesn't match the expected structure
	// Fall back to the standard transformation
	return t.transformShapeNodeWithExpectedType(&shape, expectedType)
}

// wrapVariableInNamedStruct generates a Go composite literal of the expected named struct type, mapping the variable to the appropriate field.
func (t *Transformer) wrapVariableInNamedStruct(expectedType *ast.TypeNode, variable ast.VariableNode) (goast.Expr, error) {
	goType, _ := t.transformType(*expectedType)

	// Get the struct definition to understand its fields
	def, ok := t.TypeChecker.Defs[expectedType.Ident].(ast.TypeDefNode)
	if !ok {
		return nil, fmt.Errorf("wrapVariableInNamedStruct: expected type %v is not a TypeDefNode", expectedType.Ident)
	}

	shapeExpr, ok := def.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		return nil, fmt.Errorf("wrapVariableInNamedStruct: expected type %v does not have a shape expression", expectedType.Ident)
	}

	// Create composite literal with all fields
	elts := make([]goast.Expr, 0, len(shapeExpr.Shape.Fields))

	for fieldName := range shapeExpr.Shape.Fields {
		var value goast.Expr

		// Check if this field should contain our variable
		fieldType := shapeExpr.Shape.Fields[fieldName].Type
		if fieldType != nil && fieldType.Ident == ast.TypeIdent(variable.Ident.ID) {
			// This field should contain our variable
			value = goast.NewIdent(string(variable.Ident.ID))
		} else {
			// This field needs a default value
			// For now, use zero value - this could be enhanced with field-specific defaults
			value = goast.NewIdent("0") // Default to 0, could be enhanced
		}

		elts = append(elts, &goast.KeyValueExpr{
			Key:   goast.NewIdent(fieldName),
			Value: value,
		})
	}

	return &goast.CompositeLit{
		Type: goType,
		Elts: elts,
	}, nil
}

// getShapeNode extracts a *ast.ShapeNode from an ast.Node, handling both value and pointer types
func getShapeNode(value ast.Node) (*ast.ShapeNode, bool) {
	if sn, ok := value.(*ast.ShapeNode); ok {
		return sn, true
	}
	if snVal, ok := value.(ast.ShapeNode); ok {
		return &snVal, true
	}
	return nil, false
}
