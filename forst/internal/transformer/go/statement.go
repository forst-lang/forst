package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"

	logrus "github.com/sirupsen/logrus"
)

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
		return t.transformEnsureStatement(s, stmt)
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
