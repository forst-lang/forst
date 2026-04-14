package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"

	logrus "github.com/sirupsen/logrus"
)

func (t *Transformer) transformEnsureStatement(ensureNode ast.EnsureNode, originalNode ast.Node) (goast.Stmt, error) {
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

	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function": "transformStatement",
			"stmtType": "EnsureNode",
			"stmt":     ensureNode.String(),
		}).Debug("[PINPOINT] Processing EnsureNode")
	}
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function":           "transformStatement",
			"scopeBeforeRestore": fmt.Sprintf("%v", t.currentScope()),
			"nodeType":           fmt.Sprintf("%T", originalNode),
		}).Debug("[DEBUG] Before restoreScope(EnsureNode)")
	}
	if err := t.restoreScope(originalNode); err != nil {
		return nil, fmt.Errorf("failed to restore ensure statement scope: %s", err)
	}

	stmts, err := t.transformEnsureCondition(&ensureNode)
	if err != nil {
		return nil, err
	}
	if len(stmts) == 0 {
		return nil, fmt.Errorf("transformEnsureCondition returned no statements")
	}

	exprStmt, ok := stmts[0].(*goast.ExprStmt)
	if !ok {
		return nil, fmt.Errorf("expected ExprStmt from transformEnsureCondition, got %T", stmts[0])
	}

	finalCondition := exprStmt.X
	shouldNegate := len(ensureNode.Assertion.Constraints) == 0
	for _, constraint := range ensureNode.Assertion.Constraints {
		if t.TypeChecker.IsTypeGuardConstraint(constraint.Name) {
			shouldNegate = true
			break
		}
		if (constraint.Name == "Ok" || constraint.Name == "Err") && !t.TypeChecker.IsTypeGuardConstraint(constraint.Name) {
			shouldNegate = true
			break
		}
	}
	if shouldNegate {
		finalCondition = &goast.UnaryExpr{Op: token.NOT, X: finalCondition}
	}

	finallyStmts := []goast.Stmt{}
	errorStmt := t.transformErrorStatement(fn, ensureNode)
	if ensureNode.Block != nil {
		if err := t.restoreScope(ensureNode.Block); err != nil {
			return nil, fmt.Errorf("failed to restore ensure block scope: %w", err)
		}
		for _, blockStatement := range ensureNode.Block.Body {
			if _, isReturn := blockStatement.(ast.ReturnNode); isReturn {
				continue
			}
			goStmt, err := t.transformStatement(blockStatement)
			if err != nil {
				return nil, err
			}
			finallyStmts = append(finallyStmts, goStmt)
		}
	}

	return &goast.IfStmt{
		Cond: finalCondition,
		Body: &goast.BlockStmt{List: append(finallyStmts, errorStmt)},
	}, nil
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
