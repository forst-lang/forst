package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

// transformType transforms a Forst type node into a Go type
// func (t *Transformer) transformType(tn ast.TypeNode) goast.Expr {
// 	// Look up the type alias hash for the type
// 	var typeName string
// 	if t != nil {
// 		name, err := t.getTypeAliasNameForTypeNode(tn)
// 		if err != nil {
// 			typeName = string(tn.Ident)
// 		} else {
// 			typeName = name
// 		}
// 	} else {
// 		typeName = string(tn.Ident)
// 	}
// 	return goast.NewIdent(typeName)
// }

// transformBlock transforms a block of Forst statements into Go statements
func (t *Transformer) transformBlock(block []ast.Node) *goast.BlockStmt {
	var stmts []goast.Stmt
	for _, node := range block {
		switch n := node.(type) {
		case ast.ExpressionNode:
			stmts = append(stmts, &goast.ExprStmt{
				X: t.transformExpression(n),
			})
		case ast.ReturnNode:
			stmts = append(stmts, &goast.ReturnStmt{
				Results: []goast.Expr{
					t.transformExpression(n.Value),
				},
			})
		}
	}
	return &goast.BlockStmt{
		List: stmts,
	}
}

// transformTypeGuard transforms a type guard into a Go function
func (t *Transformer) transformTypeGuard(guard ast.TypeGuardNode) (*goast.FuncDecl, error) {
	// Create function name
	guardFuncName := ast.TypeIdent(guard.Ident)

	// Create parameter list
	var params []*goast.Field

	// Transform subject parameter
	switch p := guard.Subject.(type) {
	case ast.SimpleParamNode:
		ident, err := t.transformType(p.GetType())
		if err != nil {
			return nil, fmt.Errorf("failed to transform subject parameter type: %s", err)
		}
		params = append(params, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(p.GetIdent())},
			Type:  ident,
		})
	case ast.DestructuredParamNode:
		return nil, fmt.Errorf("destructured parameters not supported in type guard")
	}

	// Transform additional parameters
	for _, param := range guard.Params {
		switch p := param.(type) {
		case ast.SimpleParamNode:
			ident, err := t.transformType(p.GetType())
			if err != nil {
				return nil, fmt.Errorf("failed to transform additional parameter type: %s", err)
			}
			params = append(params, &goast.Field{
				Names: []*goast.Ident{goast.NewIdent(p.GetIdent())},
				Type:  ident,
			})
		case ast.DestructuredParamNode:
			return nil, fmt.Errorf("destructured parameters not supported in type guard")
		}
	}

	// Transform the body into a series of if-else blocks
	var bodyStmts []goast.Stmt
	for _, node := range guard.Body {
		switch n := node.(type) {
		case *ast.IfNode:
			// Transform if condition (must be an is assertion)
			cond, ok := n.Condition.(ast.ExpressionNode)
			if !ok {
				return nil, fmt.Errorf("if condition must be an expression")
			}

			// Transform if body
			ifBody := t.transformBlock(n.Body)

			// Transform else-if blocks
			var elseIfs []goast.Stmt
			for _, elseIf := range n.ElseIfs {
				elseIfCond, ok := elseIf.Condition.(ast.ExpressionNode)
				if !ok {
					return nil, fmt.Errorf("else-if condition must be an expression")
				}
				elseIfs = append(elseIfs, &goast.IfStmt{
					Cond: t.transformExpression(elseIfCond),
					Body: t.transformBlock(elseIf.Body),
				})
			}

			// Transform else block
			var elseBody *goast.BlockStmt
			if n.Else != nil {
				elseBody = t.transformBlock(n.Else.Body)
			}

			// Add if statement to body
			bodyStmts = append(bodyStmts, &goast.IfStmt{
				Cond: t.transformExpression(cond),
				Body: ifBody,
				Else: &goast.BlockStmt{
					List: append(elseIfs, elseBody),
				},
			})

		case ast.EnsureNode:
			// Transform ensure statement into a guard
			// If the assertion fails, return false
			bodyStmts = append(bodyStmts, &goast.IfStmt{
				Cond: &goast.UnaryExpr{
					Op: token.NOT,
					X:  t.transformEnsureCondition(n),
				},
				Body: &goast.BlockStmt{
					List: []goast.Stmt{
						&goast.ReturnStmt{
							Results: []goast.Expr{
								goast.NewIdent("false"),
							},
						},
					},
				},
			})
		}
	}

	// Add final return true if all ensures passed
	bodyStmts = append(bodyStmts, &goast.ReturnStmt{
		Results: []goast.Expr{
			goast.NewIdent("true"),
		},
	})

	// Create function declaration
	return &goast.FuncDecl{
		Recv: nil,
		Name: goast.NewIdent(string(guardFuncName)),
		Type: &goast.FuncType{
			Params: &goast.FieldList{
				List: params,
			},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{
						Type: goast.NewIdent("bool"),
					},
				},
			},
		},
		Body: &goast.BlockStmt{
			List: bodyStmts,
		},
	}, nil
}
