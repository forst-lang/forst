package transformer_go

import (
	"fmt"
	"forst/pkg/ast"
	goast "go/ast"
	"go/token"
)

func (t *Transformer) transformTypeDef(node ast.TypeDefNode) *goast.GenDecl {
	return &goast.GenDecl{
		Tok: token.TYPE,
		Specs: []goast.Spec{
			&goast.TypeSpec{
				Name: &goast.Ident{
					Name: string(node.Ident),
				},
				Type: *t.transformTypeDefExpr(node.Expr),
			},
		},
	}
}

func (t *Transformer) transformTypeDefExpr(expr ast.TypeDefExpr) *goast.Expr {
	switch e := expr.(type) {
	case ast.TypeDefAssertionExpr:
		baseType := ""
		if e.Assertion.BaseType != nil {
			baseType = string(*e.Assertion.BaseType)
		}
		ident := goast.NewIdent(baseType)
		var result goast.Expr = ident
		return &result
	case ast.TypeDefBinaryExpr:
		// binaryExpr := expr.(ast.TypeDefBinaryExpr)
		// if binaryExpr.IsConjunction() {
		// 	return &goast.InterfaceType{
		// 		Methods: &goast.FieldList{
		// 			List: []*goast.Field{
		// 				{Type: *t.transformTypeDefExpr(binaryExpr.Left)},
		// 				{Type: *t.transformTypeDefExpr(binaryExpr.Right)},
		// 			},
		// 		},
		// 	}
		// } else if binaryExpr.IsDisjunction() {
		// 	return &goast.InterfaceType{
		// 		Methods: &goast.FieldList{
		// 			List: []*goast.Field{
		// 				{Type: *t.transformTypeDefExpr(binaryExpr.Left)},
		// 				{Type: *t.transformTypeDefExpr(binaryExpr.Right)},
		// 			},
		// 		},
		// 	}
		// }
		ident := goast.NewIdent("string")
		var result goast.Expr = ident
		return &result
	default:
		panic(fmt.Sprintf("Unknown type def expr: %T", expr))
	}
}
