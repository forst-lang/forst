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
		condition, err := t.transformEnsureCondition(s)
		if err != nil {
			return nil, err
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
		return &goast.ReturnStmt{
			Results: []goast.Expr{
				t.transformExpression(s.Value),
			},
		}, nil
	case ast.FunctionCallNode:
		args := make([]goast.Expr, len(s.Arguments))
		for i, arg := range s.Arguments {
			args[i] = t.transformExpression(arg)
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
			rhs := t.transformExpression(s.RValues[0])
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
			lhs[i] = t.transformExpression(lval)
		}
		rhs := make([]goast.Expr, len(s.RValues))
		for i, rval := range s.RValues {
			rhs[i] = t.transformExpression(rval)
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
