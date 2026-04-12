package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"

	logrus "github.com/sirupsen/logrus"
)

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

// zeroValueExprForASTType returns a Go expression for the zero value of a Forst type.
// Named struct types use a composite literal; builtins use getZeroValue after transformType.
func (t *Transformer) zeroValueExprForASTType(ty ast.TypeNode) (goast.Expr, error) {
	if def, ok := t.TypeChecker.Defs[ty.Ident].(ast.TypeDefNode); ok {
		if _, ok := ast.PayloadShape(def.Expr); ok {
			return t.buildZeroCompositeLiteral(&ty), nil
		}
	}
	if ty.TypeKind == ast.TypeKindUserDefined || ty.TypeKind == ast.TypeKindHashBased {
		return t.buildZeroCompositeLiteral(&ty), nil
	}
	goType, err := t.transformType(ty)
	if err != nil {
		return nil, err
	}
	return getZeroValue(goType), nil
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
	// Result(S, Error) is one Forst return type but lowers to (S, error) in Go.
	if len(returnTypes) == 1 && returnTypes[0].IsResultType() && len(returnTypes[0].TypeParams) >= 2 {
		succT := returnTypes[0].TypeParams[0]
		zeroSucc, err := t.zeroValueExprForASTType(succT)
		if err != nil {
			zeroSucc = t.buildZeroCompositeLiteral(&succT)
		}
		t.Output.EnsureImport("errors")
		assertionMsg := t.getAssertionStringForError(&stmt.Assertion)
		errExpr := &goast.CallExpr{
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
		return &goast.ReturnStmt{
			Results: []goast.Expr{zeroSucc, errExpr},
		}
	}

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
				zv, err := t.zeroValueExprForASTType(returnType)
				if err != nil {
					goType, _ := t.transformType(returnType)
					result = getZeroValue(goType)
				} else {
					result = zv
				}
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
	payload, ok := ast.PayloadShape(def.Expr)
	if !ok {
		return goast.NewIdent(valueVar)
	}
	fields := payload.Fields
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
	payload, ok := ast.PayloadShape(def.Expr)
	if !ok {
		return goast.NewIdent("nil")
	}
	fields := payload.Fields
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

		// Case 3: Result Ok/Err discriminators (not user type guards named Ok/Err)
		if !shouldNegate {
			for _, constraint := range s.Assertion.Constraints {
				if (constraint.Name == "Ok" || constraint.Name == "Err") && !t.TypeChecker.IsTypeGuardConstraint(constraint.Name) {
					shouldNegate = true
					break
				}
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

		// Result(S, F) is one Forst return type but lowers to (S, error) in Go.
		// Constructor-free success: plain `S` becomes `return succ, nil`.
		// `return g()` where `g()` is already `Result` stays a single Go return (`return g()`), not `g(), nil`.
		if len(s.Values) == 1 && len(expectedReturnTypes) == 1 && expectedReturnTypes[0].IsResultType() {
			if !t.returnValueDelegatesWholeResult(s.Values[0]) {
				succExpr, err := t.transformExpression(s.Values[0])
				if err != nil {
					return nil, err
				}
				return &goast.ReturnStmt{Results: []goast.Expr{succExpr, goast.NewIdent("nil")}}, nil
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
					for j, ret := range results {
						if ident, ok := ret.(*goast.Ident); ok {
							// PINPOINT: Log when calling buildCompositeLiteralForReturn
							if t.log != nil {
								t.log.WithFields(logrus.Fields{
									"function":     functionName,
									"returnIndex":  j,
									"expectedType": expectedType.Ident,
									"identName":    ident.Name,
								}).Debug("[PINPOINT] Calling buildCompositeLiteralForReturn for user-defined type")
							}
							goRet := t.buildCompositeLiteralForReturn(expectedType, ident.Name)
							results[j] = goRet
						}
					}
					// Function calls and other expressions are not placed in results[] by the ident→composite pass above.
					if results[i] == nil {
						valueExpr, err = t.transformExpression(value)
						if err != nil {
							return nil, err
						}
					} else {
						valueExpr = results[i]
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
			skipNilPadding := false
			// Single expression like `return g()` where g returns (T, error): do not pad with nil.
			// Go allows `return g()` as one Result holding a multi-value call; padding would produce
			// `return g(), nil` which is invalid (multi-value g() in single-value position).
			if len(s.Values) == 1 {
				if fc, ok := s.Values[0].(ast.FunctionCallNode); ok {
					if calleeSig, ok := t.TypeChecker.Functions[fc.Function.ID]; ok &&
						len(calleeSig.ReturnTypes) == len(expectedReturnTypes) {
						skipNilPadding = true
					}
				}
			}
			if !skipNilPadding {
				// Function expects more return values than provided
				// Add nil for missing error returns
				for i := len(results); i < len(expectedReturnTypes); i++ {
					expectedType := expectedReturnTypes[i]
					if expectedType.IsError() {
						results = append(results, goast.NewIdent("nil"))
					} else {
						zv, err := t.zeroValueExprForASTType(expectedType)
						if err != nil {
							return nil, fmt.Errorf("transformReturnStatement: zero value for %s: %w", expectedType.Ident, err)
						}
						results = append(results, zv)
					}
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
		if isPrintLikeBuiltinCall(s.Function) {
			args, err := t.transformPrintBuiltinCallArgs(s.Arguments)
			if err != nil {
				return nil, err
			}
			return &goast.ExprStmt{
				X: &goast.CallExpr{
					Fun:  goFunExprFromForstCallIdent(s.Function),
					Args: args,
				},
			}, nil
		}
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
		call := &goast.CallExpr{
			Fun:  goFunExprFromForstCallIdent(s.Function),
			Args: args,
		}
		// Multi-return calls (folded Result → (succ…, error), or native Go multi-return) are valid
		// Go expression statements: return values are discarded (see Go spec, Expression statements).
		return &goast.ExprStmt{X: call}, nil
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
				vn, vok := s.LValues[0].(ast.VariableNode)
				if !vok {
					return nil, fmt.Errorf("assignment: explicit type requires a simple variable on the left")
				}
				// Use the unified helper to determine the expected type
				context := &ShapeContext{
					ExpectedType: expectedType,
					VariableName: vn.Ident.String(),
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
								Names:  []*goast.Ident{goast.NewIdent(vn.Ident.String())},
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
			vn2, ok := s.LValues[0].(ast.VariableNode)
			if !ok {
				return nil, fmt.Errorf("assignment: explicit type requires a simple variable on the left")
			}
			return &goast.DeclStmt{
				Decl: &goast.GenDecl{
					Tok: token.VAR,
					Specs: []goast.Spec{
						&goast.ValueSpec{
							Names:  []*goast.Ident{goast.NewIdent(vn2.Ident.String())},
							Type:   typeExpr,
							Values: []goast.Expr{rhs},
						},
					},
				},
			}, nil
		}

		// Handle assignment without explicit types (normal assignment)
		// Result(S,F) is one Forst value but lowers to (success..., error) in Go: multi-value assignment.
		if len(s.LValues) == 1 && len(s.RValues) == 1 {
			if vn, ok := s.LValues[0].(ast.VariableNode); ok {
				if fc, ok := s.RValues[0].(ast.FunctionCallNode); ok && t.rhsCallIsFoldedResult(fc) {
					ts, err := t.TypeChecker.LookupInferredType(fc, false)
					if err != nil || len(ts) != 1 || !ts[0].IsResultType() {
						return nil, fmt.Errorf("assignment: expected Result from folded Go call")
					}
					succ := ts[0].TypeParams[0]
					var successNames []string
					if succ.IsTupleType() {
						k := len(succ.TypeParams)
						successNames = make([]string, k)
						for i := 0; i < k; i++ {
							successNames[i] = fmt.Sprintf("%s%d", string(vn.Ident.ID), i)
						}
					} else {
						successNames = []string{string(vn.Ident.ID)}
					}
					errName := string(vn.Ident.ID) + "Err"
					rhsExpr, err := t.transformExpression(fc)
					if err != nil {
						return nil, err
					}
					if t.resultLocalSplit == nil {
						t.resultLocalSplit = make(map[string]resultLocalSplit)
					}
					t.resultLocalSplit[string(vn.Ident.ID)] = resultLocalSplit{
						errGoName:      errName,
						successGoNames: successNames,
					}
					op := token.ASSIGN
					if s.IsShort {
						op = token.DEFINE
					}
					lhs := make([]goast.Expr, 0, len(successNames)+1)
					for _, n := range successNames {
						lhs = append(lhs, goast.NewIdent(n))
					}
					lhs = append(lhs, goast.NewIdent(errName))
					return &goast.AssignStmt{
						Lhs: lhs,
						Tok: op,
						Rhs: []goast.Expr{rhsExpr},
					}, nil
				}
			}
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
				varName := ""
				if vn, vok := s.LValues[0].(ast.VariableNode); vok {
					varName = vn.Ident.String()
				}
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
	case *ast.DeferNode:
		ex, err := t.transformExpression(s.Call)
		if err != nil {
			return nil, err
		}
		call, ok := ex.(*goast.CallExpr)
		if !ok {
			return nil, fmt.Errorf("defer: internal error, expected call expression, got %T", ex)
		}
		return &goast.DeferStmt{Call: call}, nil
	case *ast.GoStmtNode:
		ex, err := t.transformExpression(s.Call)
		if err != nil {
			return nil, err
		}
		call, ok := ex.(*goast.CallExpr)
		if !ok {
			return nil, fmt.Errorf("go: internal error, expected call expression, got %T", ex)
		}
		return &goast.GoStmt{Call: call}, nil
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
	case ast.CommentNode:
		return &goast.EmptyStmt{}, nil
	default:
		return &goast.EmptyStmt{}, nil
	}
}

// wrapVariableInNamedStruct generates a Go composite literal of the expected named struct type, mapping the variable to the appropriate field.
func (t *Transformer) wrapVariableInNamedStruct(expectedType *ast.TypeNode, variable ast.VariableNode) (goast.Expr, error) {
	goType, _ := t.transformType(*expectedType)

	// Get the struct definition to understand its fields
	def, ok := t.TypeChecker.Defs[expectedType.Ident].(ast.TypeDefNode)
	if !ok {
		return nil, fmt.Errorf("wrapVariableInNamedStruct: expected type %v is not a TypeDefNode", expectedType.Ident)
	}

	payload, ok := ast.PayloadShape(def.Expr)
	if !ok {
		return nil, fmt.Errorf("wrapVariableInNamedStruct: expected type %v does not have a shape or error payload", expectedType.Ident)
	}

	// Create composite literal with all fields
	elts := make([]goast.Expr, 0, len(payload.Fields))

	for fieldName := range payload.Fields {
		var value goast.Expr

		// Check if this field should contain our variable
		fieldType := payload.Fields[fieldName].Type
		if fieldType != nil && fieldType.Ident == ast.TypeIdent(variable.Ident.ID) {
			// This field should contain our variable
			value = goast.NewIdent(string(variable.Ident.ID))
		} else {
			// This field needs a default value
			// Use proper zero values based on field type
			fieldType := payload.Fields[fieldName].Type
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
