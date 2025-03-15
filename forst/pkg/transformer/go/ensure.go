package transformer_go

import (
	"forst/pkg/ast"
	goast "go/ast"
	"go/token"
)

func negateCondition(condition goast.Expr) goast.Expr {
	return &goast.UnaryExpr{
		Op: token.NOT,
		X:  condition,
	}
}

func any(conditions []goast.Expr) goast.Expr {
	if len(conditions) == 0 {
		return &goast.Ident{Name: "false"}
	}
	combined := conditions[0]
	for i := 1; i < len(conditions); i++ {
		combined = &goast.BinaryExpr{
			X:  combined,
			Op: token.LOR,
			Y:  conditions[i],
		}
	}
	return combined
}

func transformStringAssertion(ensure ast.EnsureNode) goast.Expr {
	var result []goast.Expr = []goast.Expr{}

	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case "Min":
			if len(constraint.Args) != 1 {
				panic("Min constraint requires 1 argument")
			}
			expr = &goast.BinaryExpr{
				X: &goast.CallExpr{
					Fun: &goast.SelectorExpr{
						X:   goast.NewIdent("len"),
						Sel: goast.NewIdent("s"),
					},
				},
				Op: token.LSS,
				Y:  transformExpression(constraint.Args[0]),
			}
		case "Max":
			if len(constraint.Args) != 1 {
				panic("Max constraint requires 1 argument")
			}
			expr = &goast.BinaryExpr{
				X: &goast.CallExpr{
					Fun: &goast.SelectorExpr{
						X:   goast.NewIdent("len"),
						Sel: goast.NewIdent("s"),
					},
				},
				Op: token.GTR,
				Y:  transformExpression(constraint.Args[0]),
			}
		case "HasPrefix":
			if len(constraint.Args) != 1 {
				panic("HasPrefix constraint requires 1 argument")
			}
			expr = negateCondition(&goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   goast.NewIdent("strings"),
					Sel: goast.NewIdent("HasPrefix"),
				},
				Args: []goast.Expr{
					goast.NewIdent("s"),
					transformExpression(constraint.Args[0]),
				},
			})
		default:
			panic("Unknown String constraint: " + constraint.Name)
		}

		result = append(result, expr)
	}
	return any(result)
}

func transformIntAssertion(ensure ast.EnsureNode) goast.Expr {
	var result []goast.Expr = []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case "Min":
			if len(constraint.Args) != 1 {
				panic("Min constraint requires 1 argument")
			}
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable),
				Op: token.LSS,
				Y:  transformExpression(constraint.Args[0]),
			}
		case "Max":
			if len(constraint.Args) != 1 {
				panic("Max constraint requires 1 argument")
			}
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable),
				Op: token.GTR,
				Y:  transformExpression(constraint.Args[0]),
			}
		default:
			panic("Unknown Int constraint: " + constraint.Name)
		}
		result = append(result, expr)
	}
	return any(result)
}

func transformFloatAssertion(ensure ast.EnsureNode) goast.Expr {
	var result []goast.Expr = []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case "Min":
			if len(constraint.Args) != 1 {
				panic("Min constraint requires 1 argument")
			}
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable),
				Op: token.LSS,
				Y:  transformExpression(constraint.Args[0]),
			}
		case "Max":
			if len(constraint.Args) != 1 {
				panic("Max constraint requires 1 argument")
			}
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable),
				Op: token.GTR,
				Y:  transformExpression(constraint.Args[0]),
			}
		default:
			panic("Unknown Float constraint: " + constraint.Name)
		}
		result = append(result, expr)
	}
	return any(result)
}

func transformBoolAssertion(ensure ast.EnsureNode) goast.Expr {
	var result []goast.Expr = []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case "True":
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable),
				Op: token.EQL,
				Y:  goast.NewIdent("false"),
			}
		case "False":
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable),
				Op: token.EQL,
				Y:  goast.NewIdent("true"),
			}
		default:
			panic("Unknown Bool constraint: " + constraint.Name)
		}
		result = append(result, expr)
	}
	return any(result)
}

func transformErrorAssertion(ensure ast.EnsureNode) goast.Expr {
	var result []goast.Expr = []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case "Nil":
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable),
				Op: token.NEQ,
				Y:  goast.NewIdent("nil"),
			}
		default:
			panic("Unknown Error constraint: " + constraint.Name)
		}
		result = append(result, expr)
	}
	return any(result)
}

// transformEnsure converts a Forst ensure to a Go expression
func transformEnsureCondition(ensure ast.EnsureNode) goast.Expr {
	// TODO: If BaseType is nil we need to infer it from the variable under test
	if ensure.Assertion.BaseType == nil {
		// TODO: Implement base type inference for ensure
		return &goast.Ident{Name: "false"}
	}

	switch *ensure.Assertion.BaseType {
	case ast.TypeString:
		return transformStringAssertion(ensure)
	case ast.TypeInt:
		return transformIntAssertion(ensure)
	case ast.TypeFloat:
		return transformFloatAssertion(ensure)
	case ast.TypeBool:
		return transformBoolAssertion(ensure)
	case ast.TypeError:
		return transformErrorAssertion(ensure)
	default:
		panic("Unknown base type: " + *ensure.Assertion.BaseType)
	}
}
