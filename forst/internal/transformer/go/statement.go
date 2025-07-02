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

	errorMessage := &goast.BinaryExpr{
		X: &goast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote("assertion failed: "),
		},
		Op: token.ADD,
		Y: &goast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(stmt.Assertion.String()),
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
				// For all but the last (error), emit zero value
				for i := 0; i < len(goReturnTypes)-1; i++ {
					goType, _ := t.transformType(goReturnTypes[i])
					returnTypes = append(returnTypes, getZeroValue(goType))
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
			for _, def := range t.TypeChecker.Defs {
				if tg, ok := def.(ast.TypeGuardNode); ok && tg.GetIdent() == constraint.Name {
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
		// Convert return statement
		valueExpr, err := t.transformExpression(s.Value)
		if err != nil {
			return nil, err
		}

		// Check if we need to return multiple values (e.g., (string, error))
		fnNode, err := t.closestFunction()
		if err == nil {
			if fn, ok := fnNode.(ast.FunctionNode); ok {
				// Use the type checker to get the Go return types
				goReturnTypes, err := t.TypeChecker.LookupFunctionReturnType(&fn)
				if err == nil && len(goReturnTypes) > 1 {
					// Function returns multiple values, need to add nil for error
					results := []goast.Expr{valueExpr}
					// Add nil for error (last return type)
					results = append(results, goast.NewIdent("nil"))
					return &goast.ReturnStmt{
						Results: results,
					}, nil
				}
			}
		}

		// Single return value
		return &goast.ReturnStmt{
			Results: []goast.Expr{
				valueExpr,
			},
		}, nil
	case ast.FunctionCallNode:
		args := make([]goast.Expr, len(s.Arguments))
		for i, arg := range s.Arguments {
			argExpr, err := t.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			args[i] = argExpr
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
			// Only support single variable assignment for now
			varName := s.LValues[0].Ident.String()
			// Look up the type alias hash for the type
			var typeName string
			if t != nil {
				name, err := t.getTypeAliasNameForTypeNode(*s.ExplicitTypes[0])
				if err != nil {
					typeName = string(s.ExplicitTypes[0].Ident)
				} else {
					typeName = name
				}
			} else {
				typeName = string(s.ExplicitTypes[0].Ident)
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
							Type:   goast.NewIdent(typeName),
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
			rhsExpr, err := t.transformExpression(rval)
			if err != nil {
				return nil, err
			}
			rhs[i] = rhsExpr
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
