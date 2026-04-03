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
			return &goast.Ident{Name: "false"}
		case "error":
			return &goast.Ident{Name: "nil"}
		default:
			// For struct types, return nil
			return &goast.Ident{Name: "nil"}
		}
	case *goast.StarExpr:
		// For pointer types, return nil
		return &goast.Ident{Name: "nil"}
	case *goast.ArrayType:
		// For array types, return nil
		return &goast.Ident{Name: "nil"}
	case *goast.InterfaceType:
		// For interface types, return nil
		return &goast.Ident{Name: "nil"}
	default:
		// For any other type, return nil
		return &goast.Ident{Name: "nil"}
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

// transformErrorStatement converts an ensure statement to an error return statement
func (t *Transformer) transformErrorStatement(fn ast.FunctionNode, stmt ast.EnsureNode) goast.Stmt {
	// PINPOINT: Log when this function is called
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function": "transformErrorStatement",
			"stmt":     stmt.String(),
		}).Debug("[PINPOINT] transformErrorStatement called")
	}

	functionName := string(fn.Ident.ID)

	// Get the inferred function signature from the typechecker
	var returnTypes []ast.TypeNode
	if sig, exists := t.TypeChecker.Functions[fn.Ident.ID]; exists {
		returnTypes = sig.ReturnTypes
		// PINPOINT: Log the function signature found
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function":     "transformErrorStatement",
				"functionName": functionName,
				"found":        true,
				"returnTypes":  returnTypes,
			}).Debug("[PINPOINT] Found function signature in typechecker")
		}
	} else {
		// Fallback to the raw AST node return types
		returnTypes = fn.ReturnTypes
		// PINPOINT: Log the fallback
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function":     "transformErrorStatement",
				"functionName": functionName,
				"found":        false,
				"returnTypes":  returnTypes,
			}).Debug("[PINPOINT] Using fallback return types from AST")
		}
	}

	// PINPOINT: Log function signature for error return
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function":     "transformErrorStatement",
			"functionName": functionName,
			"returnCount":  len(returnTypes),
			"returnTypes":  returnTypes,
		}).Debug("[PINPOINT] Function signature for error return")
	}

	// PINPOINT: Log each return type for debugging
	for i, retType := range returnTypes {
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function":     "transformErrorStatement",
				"functionName": functionName,
				"returnIndex":  i,
				"returnType":   retType.Ident,
				"typeKind":     retType.TypeKind,
				"isError":      retType.IsError(),
			}).Debug("[PINPOINT] Processing return type in transformErrorStatement")
		}
	}

	// For main function, use os.Exit(1)
	if t.isMainFunction() {
		t.Output.EnsureImport("os")
		return &goast.ExprStmt{
			X: &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   goast.NewIdent("os"),
					Sel: goast.NewIdent("Exit"),
				},
				Args: []goast.Expr{
					&goast.BasicLit{
						Kind:  token.INT,
						Value: "1",
					},
				},
			},
		}
	}

	// For void functions or functions with no return values, use panic
	if len(returnTypes) == 0 || (len(returnTypes) == 1 && returnTypes[0].Ident == ast.TypeVoid) {
		return &goast.ExprStmt{
			X: &goast.CallExpr{
				Fun: goast.NewIdent("panic"),
				Args: []goast.Expr{
					&goast.BasicLit{
						Kind:  token.STRING,
						Value: "\"assertion failed\"",
					},
				},
			},
		}
	}

	// Build error return values based on the function's return types
	results := make([]goast.Expr, 0, len(returnTypes))
	for i, returnType := range returnTypes {
		// PINPOINT: Log processing return type for zero value
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function":     "transformErrorStatement",
				"functionName": functionName,
				"returnIndex":  i,
				"returnType":   returnType.Ident,
				"typeKind":     returnType.TypeKind,
			}).Debug("[PINPOINT] Processing return type for zero value")
		}

		var result goast.Expr
		if returnType.IsError() {
			// For error types, return an error with the assertion message
			assertionMsg := t.getAssertionStringForError(&stmt.Assertion)

			// Ensure the errors package is imported
			t.Output.EnsureImport("errors")

			result = &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   goast.NewIdent("errors"),
					Sel: goast.NewIdent("New"),
				},
				Args: []goast.Expr{
					&goast.BasicLit{
						Kind:  token.STRING,
						Value: fmt.Sprintf("\"assertion failed: %s\"", assertionMsg),
					},
				},
			}
		} else {
			// For non-error types, always use the function's declared return type (including hash-based types)
			// Never structurally alias to a user-defined type for error returns
			if returnType.TypeKind == ast.TypeKindUserDefined || returnType.TypeKind == ast.TypeKindHashBased {
				// PINPOINT: Log when calling buildZeroCompositeLiteral for user-defined or hash-based type
				if t.log != nil {
					t.log.WithFields(logrus.Fields{
						"function":     "transformErrorStatement",
						"functionName": functionName,
						"returnIndex":  i,
						"returnType":   returnType.Ident,
						"typeKind":     returnType.TypeKind,
					}).Debug("[PINPOINT] Calling buildZeroCompositeLiteral for user-defined or hash-based type")
				}
				result = t.buildZeroCompositeLiteral(&returnType)
			} else {
				// For built-in types, use getZeroValue
				goType, _ := t.transformType(returnType)
				result = getZeroValue(goType)
			}
		}
		results = append(results, result)
	}

	return &goast.ReturnStmt{
		Results: results,
	}
}

// Helper: Build a composite literal of the expected type, mapping a variable to the first field, zero for others
func (t *Transformer) buildCompositeLiteralForReturn(expectedType *ast.TypeNode, valueVar string) goast.Expr {
	// PINPOINT: Log when this function is called
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function":     "buildCompositeLiteralForReturn",
			"expectedType": expectedType.Ident,
			"valueVar":     valueVar,
		}).Debug("[PINPOINT] buildCompositeLiteralForReturn called")
	}

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
			// Use the same logic as buildZeroCompositeLiteral for consistent zero value generation
			switch field.Type.Ident {
			case ast.TypeString:
				val = &goast.BasicLit{Kind: token.STRING, Value: "\"\""}
			case ast.TypeInt:
				val = &goast.BasicLit{Kind: token.INT, Value: "0"}
			case ast.TypeBool:
				val = goast.NewIdent("false")
			case ast.TypeFloat:
				val = &goast.BasicLit{Kind: token.FLOAT, Value: "0.0"}
			case ast.TypeError:
				val = goast.NewIdent("nil")
			default:
				// For user-defined types, use nil
				val = goast.NewIdent("nil")
			}

			// PINPOINT: Log what value is being generated for each field
			if t.log != nil {
				t.log.WithFields(logrus.Fields{
					"function":     "buildCompositeLiteralForReturn",
					"fieldName":    fieldName,
					"fieldType":    field.Type.Ident,
					"valueType":    fmt.Sprintf("%T", val),
					"valueDetails": fmt.Sprintf("%+v", val),
				}).Debug("[PINPOINT] Generated value for field in buildCompositeLiteralForReturn")
			}
		}
		goFieldName := fieldName
		if t.ExportReturnStructFields {
			goFieldName = capitalizeFirst(fieldName)
		}

		elts = append(elts, &goast.KeyValueExpr{
			Key:   goast.NewIdent(goFieldName),
			Value: val,
		})
	}
	return &goast.CompositeLit{Type: goast.NewIdent(string(expectedType.Ident)), Elts: elts}
}

// Helper: Build a zero composite literal of the expected type
func (t *Transformer) buildZeroCompositeLiteral(expectedType *ast.TypeNode) goast.Expr {
	// PINPOINT: Log when this function is called
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function":     "buildZeroCompositeLiteral",
			"expectedType": expectedType.Ident,
			"isHashBased":  expectedType.IsHashBased(),
			"typeKind":     expectedType.TypeKind,
		}).Debug("[PINPOINT] buildZeroCompositeLiteral called")
	}

	def, ok := t.TypeChecker.Defs[expectedType.Ident].(ast.TypeDefNode)
	if !ok {
		// DEBUG: Log when type definition is not found
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"expectedType": expectedType.Ident,
				"found":        false,
			}).Debug("[DEBUG] Type definition not found")
		}
		return goast.NewIdent("nil")
	}
	shapeExpr, ok := def.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		return goast.NewIdent("nil")
	}
	fields := shapeExpr.Shape.Fields
	elts := make([]goast.Expr, 0, len(fields))
	for fieldName, field := range fields {
		// DEBUG: Log field processing
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"fieldName":     fieldName,
				"fieldType":     field.Type.Ident,
				"fieldTypeKind": field.Type.TypeKind,
			}).Debug("[DEBUG] Processing field for zero value")
		}

		var val goast.Expr
		switch field.Type.Ident {
		case ast.TypeString:
			val = &goast.BasicLit{Kind: token.STRING, Value: "\"\""}
		case ast.TypeInt:
			val = &goast.BasicLit{Kind: token.INT, Value: "0"}
		case ast.TypeBool:
			val = goast.NewIdent("false")
		case ast.TypeFloat:
			val = &goast.BasicLit{Kind: token.FLOAT, Value: "0.0"}
		case ast.TypeError:
			val = goast.NewIdent("nil")
		default:
			// For user-defined types, recursively generate zero values
			if field.Type.TypeKind == ast.TypeKindUserDefined {
				// Recursively build zero value for the user-defined type
				val = t.buildZeroCompositeLiteral(field.Type)
			} else {
				// For other types, use nil
				val = goast.NewIdent("nil")
			}
		}

		// DEBUG: Log the generated zero value
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"fieldName": fieldName,
				"fieldType": field.Type.Ident,
				"zeroValue": fmt.Sprintf("%T", val),
				"zeroValueDetails": func() string {
					if basicLit, ok := val.(*goast.BasicLit); ok {
						return fmt.Sprintf("BasicLit{Kind:%s, Value:%s}", basicLit.Kind, basicLit.Value)
					}
					if ident, ok := val.(*goast.Ident); ok {
						return fmt.Sprintf("Ident{Name:%s}", ident.Name)
					}
					return fmt.Sprintf("%T", val)
				}(),
			}).Debug("[DEBUG] Generated zero value for field")
		}

		goFieldName := fieldName
		if t.ExportReturnStructFields {
			goFieldName = capitalizeFirst(fieldName)
		}

		elts = append(elts, &goast.KeyValueExpr{
			Key:   goast.NewIdent(goFieldName),
			Value: val,
		})
	}

	// DEBUG: Log the final generated composite literal
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"expectedType":  expectedType.Ident,
			"fieldCount":    len(fields),
			"generatedType": string(expectedType.Ident),
		}).Debug("[DEBUG] Generated zero composite literal")
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
		}).Debug("[PINPOINT] transformStatement called")
	}

	switch s := stmt.(type) {
	case ast.EnsureNode:
		// Track that this function has ensure statements
		fnNode, err := t.closestFunction()
		if err != nil {
			return nil, fmt.Errorf("could not find enclosing function for EnsureNode: %w", err)
		}
		fn, ok := fnNode.(ast.FunctionNode)
		if !ok {
			return nil, fmt.Errorf("enclosing node is not a FunctionNode")
		}
		t.functionsWithEnsure[string(fn.Ident.ID)] = true
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function": "transformStatement",
				"action":   "tracking function with ensure",
				"fnName":   string(fn.Ident.ID),
			}).Debug("[PINPOINT] Function has ensure statement")
		}

		// DEBUG: Log when EnsureNode is processed
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function": "transformStatement",
				"stmtType": "EnsureNode",
				"stmt":     s.String(),
			}).Debug("[PINPOINT] Processing EnsureNode")
		}
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function":           "transformStatement",
				"scopeBeforeRestore": fmt.Sprintf("%v", t.currentScope()),
				"nodeType":           fmt.Sprintf("%T", stmt),
			}).Debug("[DEBUG] Before restoreScope(EnsureNode)")
		}
		if err := t.restoreScope(stmt); err != nil {
			return nil, fmt.Errorf("failed to restore ensure statement scope: %s", err)
		}
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function":          "transformStatement",
				"scopeAfterRestore": fmt.Sprintf("%v", t.currentScope()),
				"nodeType":          fmt.Sprintf("%T", stmt),
			}).Debug("[DEBUG] After restoreScope(EnsureNode)")
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

		// Always restore the function scope before calling transformErrorStatement
		// (REMOVE THIS BLOCK)
		// if fnNode, err := t.closestFunction(); err == nil {
		// 	if err := t.restoreScope(fnNode); err != nil {
		// 		return nil, fmt.Errorf("failed to restore function scope before error statement: %s", err)
		// 	}
		// }

		// PINPOINT: Log before calling transformErrorStatement
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function":             "transformStatement",
				"nodeType":             fmt.Sprintf("%T", stmt),
				"scopeBeforeErrorStmt": fmt.Sprintf("%v", t.currentScope()),
			}).Debug("[DEBUG] Before transformErrorStatement")
		}

		// Always call transformErrorStatement for EnsureNode
		errorStmt := t.transformErrorStatement(fn, s)

		// Skip processing the Block of EnsureNode when it's being used for error handling
		// The transformErrorStatement function will handle all error return cases
		// Only process the Block if it contains non-return statements (which is rare for EnsureNode)
		if s.Block != nil {
			// Only process non-return statements in the block
			// Return statements should be handled by transformErrorStatement, not here
			for _, stmt := range s.Block.Body {
				if _, isReturn := stmt.(ast.ReturnNode); !isReturn {
					goStmt, err := t.transformStatement(stmt)
					if err != nil {
						return nil, err
					}
					finallyStmts = append(finallyStmts, goStmt)
				} else {
					// PINPOINT: Log when we skip a return statement in EnsureNode block
					if t.log != nil {
						t.log.WithFields(logrus.Fields{
							"function": "transformStatement",
							"stmtType": "EnsureNode",
							"action":   "skipping return statement in EnsureNode block",
						}).Debug("[PINPOINT] Skipping return statement in EnsureNode block - should be handled by transformErrorStatement")
					}
				}
			}
		}

		return &goast.IfStmt{
			Cond: finalCondition,
			Body: &goast.BlockStmt{
				List: append(finallyStmts, errorStmt),
			},
		}, nil
	case ast.ReturnNode:
		// Check if this function has ensure statements - if so, skip normal return processing
		var functionName string
		var hasEnsure bool
		fnNode, err := t.closestFunction()
		if err != nil {
			return nil, fmt.Errorf("could not find enclosing function for ReturnNode: %w", err)
		}
		fn, ok := fnNode.(ast.FunctionNode)
		if !ok {
			return nil, fmt.Errorf("enclosing node is not a FunctionNode")
		}
		functionName = string(fn.Ident.ID)
		hasEnsure = t.functionsWithEnsure[functionName]
		if hasEnsure && t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function": "transformStatement",
				"action":   "skipping return statement for function with ensure",
				"fnName":   functionName,
			}).Debug("[PINPOINT] Skipping return statement - function has ensure statements")
		}

		// If function has ensure statements, modify return processing to handle ensure statements
		if hasEnsure {
			// For functions with ensure statements, the normal return should be the success path
			// The error path is already handled by transformErrorStatement
			// So we need to wrap this return in an else block or similar
			if t.log != nil {
				t.log.WithFields(logrus.Fields{
					"function": "transformStatement",
					"action":   "processing return for function with ensure",
					"fnName":   functionName,
				}).Debug("[PINPOINT] Processing return for function with ensure statements")
			}
		}

		// DEBUG: Log when ReturnNode is processed
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function": "transformStatement",
				"stmtType": "ReturnNode",
				"stmt":     s.String(),
			}).Debug("[PINPOINT] Processing ReturnNode")
		}
		// DEBUG: Log entry
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function": "transformReturnStatement",
				"values":   len(s.Values),
			}).Debug("transformReturnStatement called")
		}

		// Find the current function node and its inferred return types
		var expectedReturnTypes []ast.TypeNode
		fnNode, err = t.closestFunction()
		if err != nil {
			return nil, fmt.Errorf("could not find enclosing function for transformReturnStatement: %w", err)
		}
		fn, ok = fnNode.(ast.FunctionNode)
		if !ok {
			return nil, fmt.Errorf("enclosing node is not a FunctionNode")
		}
		functionName = string(fn.Ident.ID)
		// DEBUG: Log function lookup
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function":      "transformReturnStatement",
				"functionName":  fn.Ident.ID,
				"functionFound": true,
			}).Debug("Function node found")
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
				}).Debug("Function signature for return statement")
			}
		} else {
			// DEBUG: Log missing signature
			if t.log != nil {
				t.log.WithFields(logrus.Fields{
					"function":       "transformReturnStatement",
					"functionName":   fn.Ident.ID,
					"hasSig":         ok,
					"sigReturnTypes": len(sig.ReturnTypes),
				}).Debug("No function signature found")
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
				}).Debug("[PINPOINT] Mapping AST node to Go type for return value")
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
					}).Debug("[PINPOINT] Found user-defined expected type for return value")
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
						}).Debug("[PINPOINT] Wrapping variable in named struct")
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
							}).Debug("[PINPOINT] Wrapping shape (robust) in named struct")
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
						}).Debug("[PINPOINT] Unhandled value type in return statement switch")
					}
				}
			}

			var valueExpr goast.Expr
			var err error

			if i < len(expectedReturnTypes) {
				expectedType := &expectedReturnTypes[i]
				if expectedType.TypeKind == ast.TypeKindUserDefined {
					// PINPOINT: Log when processing user-defined return types
					if t.log != nil {
						t.log.WithFields(logrus.Fields{
							"function":     functionName,
							"returnIndex":  i,
							"expectedType": expectedType.Ident,
							"resultsLen":   len(results),
						}).Debug("[PINPOINT] Processing user-defined return type in transformReturnStatement")
					}
					for i, ret := range results {
						if ident, ok := ret.(*goast.Ident); ok {
							// PINPOINT: Log when calling buildCompositeLiteralForReturn
							if t.log != nil {
								t.log.WithFields(logrus.Fields{
									"function":     functionName,
									"returnIndex":  i,
									"expectedType": expectedType.Ident,
									"identName":    ident.Name,
								}).Debug("[PINPOINT] Calling buildCompositeLiteralForReturn for user-defined type")
							}
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
							}).Debug("[PINPOINT] Processing shape literal without user-defined expected type")
						}

						context := &ShapeContext{
							ExpectedType: expectedType,
							ReturnIndex:  i,
						}
						fnNode, err := t.closestFunction()
						if err == nil {
							fn, ok := fnNode.(ast.FunctionNode)
							if ok {
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
							}).Debug("[PINPOINT] Processing struct literal in return statement")
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
							}).Debug("[PINPOINT] Processing non-shape return value")
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
					}).Debug("[PINPOINT] No expected type available for return value")
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
			fnNode, err := t.closestFunction()
			if err == nil {
				fn, ok := fnNode.(ast.FunctionNode)
				if ok {
					t.log.WithFields(logrus.Fields{
						"function":     "transformReturnStatement",
						"functionName": string(fn.Ident.ID),
						"returnTypes":  expectedReturnTypes,
					}).Debug("Function signature for return statement")
				}
			}
		}

		return &goast.ReturnStmt{
			Results: results,
		}, nil

	case ast.FunctionCallNode:
		// Look up parameter types for the function
		paramTypes := make([]ast.TypeNode, len(s.Arguments))
		fnNode, err := t.closestFunction()
		if err != nil {
			return nil, fmt.Errorf("could not find enclosing function for FunctionCallNode: %w", err)
		}
		fn, ok := fnNode.(ast.FunctionNode)
		if !ok {
			return nil, fmt.Errorf("enclosing node is not a FunctionNode")
		}
		if sig, ok := t.TypeChecker.Functions[fn.Ident.ID]; ok && len(sig.Parameters) == len(s.Arguments) {
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
					VariableName: s.LValues[0].Ident.String(),
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
								Names:  []*goast.Ident{goast.NewIdent(s.LValues[0].Ident.String())},
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
							Names:  []*goast.Ident{goast.NewIdent(s.LValues[0].Ident.String())},
							Type:   typeExpr,
							Values: []goast.Expr{rhs},
						},
					},
				},
			}, nil
		}

		// Handle assignment without explicit types (normal assignment)
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
	case *ast.IfNode:
		return t.transformIfNode(s)
	case *ast.ForNode:
		return t.transformForNode(s)
	case *ast.BreakNode:
		bs := &goast.BranchStmt{Tok: token.BREAK}
		if s.Label != nil {
			bs.Label = goast.NewIdent(string(s.Label.ID))
		}
		return bs, nil
	case *ast.ContinueNode:
		cs := &goast.BranchStmt{Tok: token.CONTINUE}
		if s.Label != nil {
			cs.Label = goast.NewIdent(string(s.Label.ID))
		}
		return cs, nil
	case ast.UnaryExpressionNode:
		if s.Operator == ast.TokenPlusPlus || s.Operator == ast.TokenMinusMinus {
			v, ok := s.Operand.(ast.VariableNode)
			if !ok {
				return nil, fmt.Errorf("++/-- only applies to variables")
			}
			tok := token.INC
			if s.Operator == ast.TokenMinusMinus {
				tok = token.DEC
			}
			return &goast.IncDecStmt{
				X:   goast.NewIdent(string(v.Ident.ID)),
				Tok: tok,
			}, nil
		}
		ex, err := t.transformExpression(s)
		if err != nil {
			return nil, err
		}
		return &goast.ExprStmt{X: ex}, nil
	case ast.VariableNode:
		// This case is now handled by transformReturnStatement
		return nil, fmt.Errorf("transformStatement: VariableNode should not be directly transformed here")
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
		}).Debug("wrapShapeInNamedStructIfNeeded called")
	}

	// First, try to use the existing wrapInNamedStruct logic
	expr, err := t.wrapInNamedStruct(expectedType, shape)
	if err == nil {
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"function": "wrapShapeInNamedStructIfNeeded",
				"result":   "used wrapInNamedStruct",
			}).Debug("wrapShapeInNamedStructIfNeeded: used wrapInNamedStruct")
		}
		return expr, nil
	}

	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function": "wrapShapeInNamedStructIfNeeded",
			"error":    err.Error(),
		}).Debug("wrapShapeInNamedStructIfNeeded: wrapInNamedStruct failed, trying field matching")
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
		}).Debug("wrapShapeInNamedStructIfNeeded: comparing field counts")
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
				}).Debug("wrapShapeInNamedStructIfNeeded: field names match, wrapping")
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
		}).Debug("wrapShapeInNamedStructIfNeeded: falling back to standard transformation")
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
			// Use proper zero values based on field type
			fieldType := shapeExpr.Shape.Fields[fieldName].Type
			if fieldType != nil {
				switch fieldType.Ident {
				case ast.TypeString:
					value = &goast.BasicLit{Kind: token.STRING, Value: "\"\""}
				case ast.TypeInt:
					value = &goast.BasicLit{Kind: token.INT, Value: "0"}
				case ast.TypeBool:
					value = goast.NewIdent("false")
				case ast.TypeFloat:
					value = &goast.BasicLit{Kind: token.FLOAT, Value: "0.0"}
				case ast.TypeError:
					value = goast.NewIdent("nil")
				default:
					// For user-defined types, use nil
					value = goast.NewIdent("nil")
				}
			} else {
				// Fallback to 0 if field type is unknown
				value = goast.NewIdent("0")
			}
		}

		goFieldName := fieldName
		if t.ExportReturnStructFields {
			goFieldName = capitalizeFirst(fieldName)
		}

		elts = append(elts, &goast.KeyValueExpr{
			Key:   goast.NewIdent(goFieldName),
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
