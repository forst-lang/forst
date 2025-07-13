package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"

	logrus "github.com/sirupsen/logrus"
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
			expr, err := t.transformExpression(n)
			if err != nil {
				t.log.WithError(err).Error("Failed to transform expression")
				continue
			}
			stmts = append(stmts, &goast.ExprStmt{
				X: expr,
			})
		case ast.ReturnNode:
			// Transform all return values
			results := make([]goast.Expr, len(n.Values))
			for i, value := range n.Values {
				expr, err := t.transformExpression(value)
				if err != nil {
					t.log.WithError(err).Error("Failed to transform expression")
					continue
				}
				results[i] = expr
			}
			stmts = append(stmts, &goast.ReturnStmt{
				Results: results,
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
			return nil, fmt.Errorf("DestructuredParamNode not supported in transformTypeGuardParams")
		}

		var ident *goast.Ident
		if t != nil {
			name, err := t.TypeChecker.GetAliasedTypeName(paramType)
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
	// Create function name with G_ prefix
	guardHash, err := t.TypeChecker.Hasher.HashNode(guard)
	if err != nil {
		return nil, fmt.Errorf("failed to hash guard: %s", err)
	}
	guardIdent := guardHash.ToGuardIdent()

	t.log.WithFields(logrus.Fields{
		"guard":      guard.Ident,
		"function":   "transformTypeGuard",
		"guardIdent": guardIdent,
	}).Debug("Transforming type guard")

	// Check if this is a type-level type guard (only contains type-level assertions)
	if t.isTypeLevelTypeGuard(guard) {
		t.log.WithFields(logrus.Fields{
			"guard":    guard.Ident,
			"function": "transformTypeGuard",
		}).Debug("Skipping function generation for type-level type guard")
		return nil, nil // Return nil to indicate no function should be generated
	}

	// Transform subject parameter
	subjectParam, err := t.transformTypeGuardParams([]ast.ParamNode{guard.Subject})
	if err != nil {
		return nil, fmt.Errorf("failed to transform subject parameter: %s", err)
	}

	// Create parameter list
	additionalParams, err := t.transformTypeGuardParams(guard.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to transform type guard parameters: %s", err)
	}

	params := append(subjectParam.List, additionalParams.List...)

	// Transform the body into a series of if-else blocks
	var bodyStmts []goast.Stmt
	for _, node := range guard.Body {
		// Ensure the type guard parameter scope is active
		if err := t.restoreScope(guard); err != nil {
			return nil, fmt.Errorf("failed to restore type guard parameter scope: %s", err)
		}
		switch n := node.(type) {
		case *ast.IfNode:
			if err := t.restoreScope(*n); err != nil {
				return nil, fmt.Errorf("failed to restore if scope in type guard: %s", err)
			}

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
				if err := t.restoreScope(elseIf); err != nil {
					return nil, fmt.Errorf("failed to restore else-if scope in type guard: %s", err)
				}

				elseIfCond, ok := elseIf.Condition.(ast.ExpressionNode)
				if !ok {
					return nil, fmt.Errorf("else-if condition must be an expression")
				}
				elseIfCondExpr, err := t.transformExpression(elseIfCond)
				if err != nil {
					return nil, fmt.Errorf("failed to transform else-if condition: %s", err)
				}
				elseIfs = append(elseIfs, &goast.IfStmt{
					Cond: elseIfCondExpr,
					Body: t.transformBlock(elseIf.Body),
				})
			}

			// Transform else block
			var elseBody *goast.BlockStmt
			if n.Else != nil {
				if err := t.restoreScope(*n.Else); err != nil {
					return nil, fmt.Errorf("failed to restore else scope in type guard: %s", err)
				}

				elseBody = t.transformBlock(n.Else.Body)
			}

			// Add if statement to body
			condExpr, err := t.transformExpression(cond)
			if err != nil {
				return nil, fmt.Errorf("failed to transform if condition: %s", err)
			}
			bodyStmts = append(bodyStmts, &goast.IfStmt{
				Cond: condExpr,
				Body: ifBody,
				Else: &goast.BlockStmt{
					List: append(elseIfs, elseBody),
				},
			})

		case ast.EnsureNode:
			if err := t.restoreScope(n); err != nil {
				return nil, fmt.Errorf("failed to restore ensure statement scope in type guard: %s", err)
			}

			// Transform ensure statement into a boolean expression
			// For type guards, we want to return true if the condition is met
			condStmts, err := t.transformTypeGuardEnsure(&n)
			if err != nil {
				return nil, fmt.Errorf("failed to transform ensure condition in type guard: %s", err)
			}
			t.log.WithFields(logrus.Fields{
				"ensure":   n,
				"stmts":    condStmts,
				"function": "transformTypeGuard",
			}).Trace("Transformed ensure condition")

			// If the condition is not met, return false
			// Use the first statement's expression as the condition
			if len(condStmts) == 0 {
				return nil, fmt.Errorf("no statements generated from ensure condition")
			}
			exprStmt, ok := condStmts[0].(*goast.ExprStmt)
			if !ok {
				return nil, fmt.Errorf("first statement is not an expression statement")
			}

			bodyStmts = append(bodyStmts, &goast.IfStmt{
				Cond: exprStmt.X,
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

	// Add default return true at the end
	bodyStmts = append(bodyStmts, &goast.ReturnStmt{
		Results: []goast.Expr{
			goast.NewIdent("true"),
		},
	})

	// Create the function declaration
	decl := &goast.FuncDecl{
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
	}

	t.log.WithFields(logrus.Fields{
		"guard":      guard.Ident,
		"function":   "transformTypeGuard",
		"guardIdent": guardIdent,
		"declName":   decl.Name.Name,
		"params":     len(params),
		"bodyStmts":  len(bodyStmts),
	}).Debug("Created type guard function declaration")

	return decl, nil
}

// isTypeLevelTypeGuard checks if a type guard contains only type-level assertions
func (t *Transformer) isTypeLevelTypeGuard(guard ast.TypeGuardNode) bool {
	for _, node := range guard.Body {
		switch n := node.(type) {
		case ast.EnsureNode:
			// Check if all constraints in the ensure statement are type-level
			for _, constraint := range n.Assertion.Constraints {
				if !t.isTypeLevelConstraint(constraint) {
					return false
				}
			}
		default:
			// If there are any non-ensure nodes, it's not purely type-level
			return false
		}
	}
	return true
}

// isTypeLevelConstraint checks if a constraint is type-level (like "is" operator)
func (t *Transformer) isTypeLevelConstraint(constraint ast.ConstraintNode) bool {
	// Type-level constraints are those that can't be transformed into runtime code
	// Currently, this includes the "is" operator for shape field assertions
	return constraint.Name == "is"
}
