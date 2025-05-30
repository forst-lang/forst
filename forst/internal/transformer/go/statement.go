package transformergo

import (
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
func (t *Transformer) transformStatement(stmt ast.Node) goast.Stmt {
	switch s := stmt.(type) {
	case ast.EnsureNode:
		// Convert ensure to if statement with panic
		condition := t.transformEnsureCondition(s)

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
				if tg, ok := def.(*ast.TypeGuardNode); ok && string(tg.Ident) == constraint.Name {
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
			t.pushScope(s.Block)
			for _, stmt := range s.Block.Body {
				goStmt := t.transformStatement(stmt)
				finallyStmts = append(finallyStmts, goStmt)
			}
			t.popScope()
		}

		errorStmt := t.transformErrorStatement(s)

		return &goast.IfStmt{
			Cond: finalCondition,
			Body: &goast.BlockStmt{
				List: append(finallyStmts, errorStmt),
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
			rhs := transformExpression(s.RValues[0])
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
			}
		}
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
