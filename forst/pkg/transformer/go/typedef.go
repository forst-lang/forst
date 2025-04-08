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

func (t *Transformer) transformAssertionType(assertion *ast.AssertionNode) *goast.Expr {
	result := goast.StructType{
		Fields: &goast.FieldList{
			List: []*goast.Field{
				// t.transformAssertion(assertion),
			},
		},
	}
	var expr goast.Expr = &result
	return &expr
}

func (t *Transformer) transformShapeFieldType(field ast.ShapeFieldNode) *goast.Expr {
	if field.Assertion != nil {
		println(fmt.Sprintf("transformShapeFieldType, assertion: %s", *field.Assertion))
		return t.transformAssertionType(field.Assertion)
	}
	if field.Shape != nil {
		println(fmt.Sprintf("transformShapeFieldType, shape: %s", *field.Shape))
		return t.transformShapeType(field.Shape)
	}
	panic(fmt.Sprintf("Shape field has neither assertion nor shape: %T", field))
}

func (t *Transformer) transformShapeType(shape *ast.ShapeNode) *goast.Expr {
	fields := []*goast.Field{}
	for name, field := range shape.Fields {
		fieldType := *t.transformShapeFieldType(field)
		fields = append(fields, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(name)},
			Type:  fieldType,
		})
	}
	result := goast.StructType{
		Fields: &goast.FieldList{
			List: fields,
		},
	}
	var expr goast.Expr = &result
	return &expr
}

func (t *Transformer) transformTypeDefExpr(expr ast.TypeDefExpr) *goast.Expr {
	switch e := expr.(type) {
	case ast.TypeDefAssertionExpr:
		baseType := ""
		if e.Assertion.BaseType != nil {
			baseType = string(*e.Assertion.BaseType)
		}

		if baseType == "trpc.Mutation" || baseType == "trpc.Query" {
			fields := []*goast.Field{
				{
					Names: []*goast.Ident{goast.NewIdent("ctx")},
					Type:  &goast.StructType{Fields: &goast.FieldList{}},
				},
			}

			for _, constraint := range e.Assertion.Constraints {
				if constraint.Name == "Input" && len(constraint.Args) > 0 {
					arg := constraint.Args[0]
					if shape := arg.Shape; shape != nil {
						inputField := goast.Field{
							Names: []*goast.Ident{goast.NewIdent("input")},
							Type:  *t.transformShapeType(shape),
						}
						fields = append(fields, &inputField)
					}
				}
			}

			result := goast.StructType{
				Fields: &goast.FieldList{
					List: fields,
				},
			}
			var expr goast.Expr = &result
			return &expr
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
