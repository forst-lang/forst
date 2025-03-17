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

		errorMsg := "assertion failed: " + s.Assertion.String()
		if s.Error != nil {
			errorMsg = (*s.Error).String()
		}

		finallyStmts := []goast.Stmt{}

		if s.Block != nil {
			t.pushScope(s.Block)
			for _, stmt := range s.Block.Body {
				finallyStmts = append(finallyStmts, t.transformStatement(stmt))
			}
			t.popScope()
		}
		return &goast.IfStmt{
			Cond: condition,
			Body: &goast.BlockStmt{
				List: append(finallyStmts, &goast.ExprStmt{
					X: &goast.CallExpr{
						Fun: goast.NewIdent("panic"),
						Args: []goast.Expr{
							&goast.BasicLit{
								Kind:  token.STRING,
								Value: strconv.Quote(errorMsg),
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
	default:
		return &goast.EmptyStmt{}
	}
}
