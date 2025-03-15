package transformer_go

import (
	"forst/pkg/ast"
	goast "go/ast"
	"go/token"
)

// transformEnsure converts a Forst ensure to a Go expression
func transformEnsureCondition(ensure ast.EnsureNode) goast.Expr {
	if ensure.Assertion.BaseType != nil && *ensure.Assertion.BaseType == "String" {
		var result goast.Expr = &goast.Ident{Name: "true"}

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
					Op: token.GEQ,
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
					Op: token.LEQ,
					Y:  transformExpression(constraint.Args[0]),
				}
			case "HasPrefix":
				if len(constraint.Args) != 1 {
					panic("HasPrefix constraint requires 1 argument")
				}
				expr = &goast.CallExpr{
					Fun: &goast.SelectorExpr{
						X:   goast.NewIdent("strings"),
						Sel: goast.NewIdent("HasPrefix"),
					},
					Args: []goast.Expr{
						goast.NewIdent("s"),
						transformExpression(constraint.Args[0]),
					},
				}
			default:
				panic("Unknown String constraint: " + constraint.Name)
			}

			result = &goast.BinaryExpr{
				X:  result,
				Op: token.LAND,
				Y:  expr,
			}
		}
		return result
	}

	return &goast.Ident{Name: "true"}
}
