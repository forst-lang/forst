package transformer

import (
	"forst/pkg/ast"
	goast "go/ast"
	"go/token"
	"strconv"
)

// TransformForstToGo converts a Forst AST to a Go AST
func TransformForstToGo(forstAST ast.FunctionNode) *goast.FuncDecl {
	// Create function parameters
	params := &goast.FieldList{
		List: []*goast.Field{},
	}
	
	for _, param := range forstAST.Params {
		params.List = append(params.List, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(param.Name)},
			Type:  goast.NewIdent(param.Type),
		})
	}
	
	// Create function return type
	var results *goast.FieldList
	if !forstAST.ReturnType.IsImplicit() {
		results = &goast.FieldList{
			List: []*goast.Field{
				{
					Type: goast.NewIdent(forstAST.ReturnType.Name),
				},
			},
		}
	}
	
	// Create function body statements
	stmts := []goast.Stmt{}
	
	for _, node := range forstAST.Body {
		switch n := node.(type) {
		case ast.EnsureNode:
			// Convert ensure to if statement with panic
			condition := parseExpr(n.Condition)
			notCondition := &goast.UnaryExpr{
				Op: token.NOT,
				X:  condition,
			}
			
			errorMsg := "assertion failed: " + n.Condition
			if n.ErrorType != "" {
				errorMsg = n.ErrorType
			}
			
			stmts = append(stmts, &goast.IfStmt{
				Cond: notCondition,
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
			})
			
		case ast.ReturnNode:
			// Convert return statement
			stmts = append(stmts, &goast.ReturnStmt{
				Results: []goast.Expr{
					parseExpr(n.Value),
				},
			})
		}
	}
	
	// Create the function declaration
	return &goast.FuncDecl{
		Name: goast.NewIdent(forstAST.Name),
		Type: &goast.FuncType{
			Params:  params,
			Results: results,
		},
		Body: &goast.BlockStmt{
			List: stmts,
		},
	}
}

// Helper function to parse expressions from strings
// This is a simplified version - in a real implementation,
// you would need a proper expression parser
func parseExpr(expr string) goast.Expr {
	// For simplicity, just return an identifier
	// In a real implementation, you would parse the expression
	return goast.NewIdent(expr)
} 