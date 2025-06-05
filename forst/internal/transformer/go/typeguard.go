package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"

	log "github.com/sirupsen/logrus"
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

func (t *Transformer) transformTypeGuardParams(params []ast.ParamNode) (*goast.FieldList, error) {

	fields := &goast.FieldList{
		List: []*goast.Field{},
	}

	for _, param := range params {
		var paramName string
		var paramType ast.TypeNode

		switch p := param.(type) {
		case ast.SimpleParamNode:
			paramName = string(p.Ident.ID)
			paramType = p.Type
		case ast.DestructuredParamNode:
			// Handle destructured params if needed
			continue
		}

		var ident *goast.Ident
		if t != nil {
			name, err := t.getTypeAliasNameForTypeNode(paramType)
			if err != nil {
				return nil, fmt.Errorf("failed to get type alias name: %s", err)
			}
			ident = goast.NewIdent(name)
		} else {
			ident = goast.NewIdent(string(paramType.Ident))
		}
		fields.List = append(fields.List, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(paramName)},
			Type:  ident,
		})
	}

	return fields, nil
}

// transformTypeGuard transforms a type guard into a Go function
func (t *Transformer) transformTypeGuard(guard ast.TypeGuardNode) (*goast.FuncDecl, error) {
	t.pushScope(guard)

	// Create function name
	guardIdent := t.TypeChecker.Hasher.HashNode(guard).ToGuardIdent()

	// Transform subject parameter
	subjectParam, err := t.transformTypeGuardParams([]ast.ParamNode{guard.Subject})
	if err != nil {
		t.popScope()
		return nil, fmt.Errorf("failed to transform subject parameter: %s", err)
	}

	// Create parameter list
	additionalParams, err := t.transformTypeGuardParams(guard.Params)
	if err != nil {
		t.popScope()
		return nil, fmt.Errorf("failed to transform type guard parameters: %s", err)
	}

	params := append(subjectParam.List, additionalParams.List...)

	// Transform the body into a series of if-else blocks
	var bodyStmts []goast.Stmt
	for _, node := range guard.Body {
		switch n := node.(type) {
		case *ast.IfNode:
			// Transform if condition (must be an is assertion)
			cond, ok := n.Condition.(ast.ExpressionNode)
			if !ok {
				t.popScope()
				return nil, fmt.Errorf("if condition must be an expression")
			}

			// Transform if body
			ifBody := t.transformBlock(n.Body)

			// Transform else-if blocks
			var elseIfs []goast.Stmt
			for _, elseIf := range n.ElseIfs {
				elseIfCond, ok := elseIf.Condition.(ast.ExpressionNode)
				if !ok {
					t.popScope()
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
			condExpr := t.transformEnsureCondition(n)
			log.Tracef("Transformed ensure condition: %+v (go expr: %s)", n, condExpr)

			// Transform ensure statement into a guard
			// If the assertion fails, return false
			bodyStmts = append(bodyStmts, &goast.IfStmt{
				Cond: t.transformEnsureCondition(n),
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

	t.popScope()

	// Create function declaration
	return &goast.FuncDecl{
		Recv: nil,
		Name: goast.NewIdent(string(guardIdent)),
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
