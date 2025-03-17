package transformer_go

import (
	"forst/pkg/ast"
	goast "go/ast"
	"go/token"
	"strconv"
)

// transformStatement converts a Forst statement to a Go statement
func (t *Transformer) transformStatement(stmt ast.Node) goast.Stmt {
	switch s := stmt.(type) {
	case ast.EnsureNode:
		// Convert ensure to if statement with panic
		condition := t.transformEnsureCondition(s)

		var errorExpr goast.Expr = &goast.BinaryExpr{
			X: &goast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote("assertion failed: "),
			},
			Op: token.ADD,
			Y: &goast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(s.Assertion.String()),
			},
		}
		if s.Error != nil {
			if errVar, ok := (*s.Error).(ast.EnsureErrorVar); ok {
				errorExpr = &goast.Ident{
					Name: string(errVar),
				}
			} else {
				errorExpr = &goast.BasicLit{
					Kind:  token.STRING,
					Value: strconv.Quote((*s.Error).String()),
				}
			}
		}

		finallyStmts := []goast.Stmt{}

		if s.Block != nil {
			t.pushScope(s.Block)
			for _, stmt := range s.Block.Body {
				goStmt := t.transformStatement(stmt)
				finallyStmts = append(finallyStmts, goStmt)
			}
			t.popScope()
		}

		isMainFunction := t.isMainFunction()

		if isMainFunction {
			return &goast.IfStmt{
				Cond: condition,
				Body: &goast.BlockStmt{
					List: append(finallyStmts, &goast.ExprStmt{
						X: &goast.CallExpr{
							Fun: goast.NewIdent("panic"),
							Args: []goast.Expr{
								errorExpr,
							},
						},
					}),
				},
			}
		}

		return &goast.IfStmt{
			Cond: condition,
			Body: &goast.BlockStmt{
				List: append(finallyStmts, &goast.ReturnStmt{
					Results: []goast.Expr{
						&goast.CallExpr{
							Fun: &goast.SelectorExpr{
								X:   goast.NewIdent("errors"),
								Sel: goast.NewIdent("New"),
							},
							Args: []goast.Expr{
								errorExpr,
							},
						},
					},
				}),
			},
		}
	case ast.ReturnNode:
		// Convert return statement
		return &goast.ReturnStmt{
			Results: []goast.Expr{
				transformExpression(s.Value),
			},
		}
	case ast.FunctionCallNode:
		args := make([]goast.Expr, len(s.Arguments))
		for i, arg := range s.Arguments {
			args[i] = transformExpression(arg)
		}
		return &goast.ExprStmt{
			X: &goast.CallExpr{
				Fun:  goast.NewIdent(s.Function.String()),
				Args: args,
			},
		}
	case ast.AssignmentNode:
		lhs := make([]goast.Expr, len(s.LValues))
		for i, lval := range s.LValues {
			lhs[i] = transformExpression(lval)
		}
		rhs := make([]goast.Expr, len(s.RValues))
		for i, rval := range s.RValues {
			rhs[i] = transformExpression(rval)
		}
		operator := token.ASSIGN
		if s.IsShort {
			operator = token.DEFINE
		}
		return &goast.AssignStmt{
			Lhs: lhs,
			Tok: operator,
			Rhs: rhs,
		}
	default:
		return &goast.EmptyStmt{}
	}
}
