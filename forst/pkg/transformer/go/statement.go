package transformer_go

import (
	"forst/pkg/ast"
	goast "go/ast"
	"go/token"
	"strconv"
)

// transformStatement converts a Forst statement to a Go statement
func transformStatement(stmt ast.Node) goast.Stmt {
	switch s := stmt.(type) {
	case ast.EnsureNode:
		// Convert ensure to if statement with panic
		condition := transformEnsureCondition(s)

		errorMsg := "assertion failed: " + s.Assertion.String()
		if s.Error != nil {
			errorMsg = (*s.Error).String()
		}

		return &goast.IfStmt{
			Cond: condition,
			Body: &goast.BlockStmt{
				List: []goast.Stmt{
					&goast.ExprStmt{
						X: &goast.CallExpr{
							Fun: goast.NewIdent("panic"),
							Args: []goast.Expr{
								&goast.BasicLit{
									Kind:  token.STRING,
									Value: strconv.Quote(errorMsg),
								},
							},
						},
					},
				},
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
