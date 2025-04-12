package transformer_go

import (
	"fmt"
	"forst/pkg/ast"
	goast "go/ast"
	"go/token"

	log "github.com/sirupsen/logrus"
)

func (t *Transformer) transformTypeDef(node ast.TypeDefNode) (*goast.GenDecl, error) {
	expr, err := t.transformTypeDefExpr(node.Expr)
	if err != nil {
		log.Error(fmt.Errorf("failed to transform type def expr during transformation: %s", err))
		return nil, err
	}
	return &goast.GenDecl{
		Tok: token.TYPE,
		Specs: []goast.Spec{
			&goast.TypeSpec{
				Name: &goast.Ident{
					Name: string(node.Ident),
				},
				Type: *expr,
			},
		},
	}, nil
}

func (t *Transformer) transformAssertionType(assertion *ast.AssertionNode) (*goast.Expr, error) {
	log.Debug(fmt.Sprintf("transformAssertionType, assertion: %s", *assertion))
	assertionType, err := t.TypeChecker.LookupAssertionType(assertion, t.currentScope)
	if assertionType == nil {
		log.Trace("assertionType: nil")
	} else {
		log.Trace(fmt.Sprintf("assertionType: %s", *assertionType))
	}
	if err != nil {
		err = fmt.Errorf("failed to lookup assertion type during transformation: %w", err)
		log.WithError(err).Error("transforming assertion type failed")
		return nil, err
	}
	var expr goast.Expr = goast.NewIdent(string(assertionType.Ident))
	return &expr, nil
}

func (t *Transformer) transformShapeFieldType(field ast.ShapeFieldNode) (*goast.Expr, error) {
	if field.Assertion != nil {
		log.Trace(fmt.Sprintf("transformShapeFieldType, assertion: %s", *field.Assertion))
		expr, err := t.transformAssertionType(field.Assertion)
		if err != nil {
			err = fmt.Errorf("failed to transform assertion type during transformation: %w", err)
			log.WithError(err).Error("transforming assertion type failed")
			return nil, err
		}
		return expr, nil
	}
	if field.Shape != nil {
		log.Trace(fmt.Sprintf("transformShapeFieldType, shape: %s", *field.Shape))
		expr, err := t.transformShapeType(field.Shape)
		if err != nil {
			err = fmt.Errorf("failed to transform shape type during transformation: %w", err)
			log.WithError(err).Error("transforming shape type failed")
			return nil, err
		}
		return expr, nil
	}
	return nil, fmt.Errorf("shape field has neither assertion nor shape: %T", field)
}

func (t *Transformer) transformShapeType(shape *ast.ShapeNode) (*goast.Expr, error) {
	fields := []*goast.Field{}
	for name, field := range shape.Fields {
		fieldType, err := t.transformShapeFieldType(field)
		if err != nil {
			err = fmt.Errorf("failed to transform shape field type during transformation: %w", err)
			log.WithError(err).Error("transforming shape field type failed")
			return nil, err
		}
		fields = append(fields, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(name)},
			Type:  *fieldType,
		})
	}
	result := goast.StructType{
		Fields: &goast.FieldList{
			List: fields,
		},
	}
	var expr goast.Expr = &result
	return &expr, nil
}

func transformTypeIdent(ident ast.TypeIdent) *goast.Ident {
	switch ident {
	case ast.TypeString:
		return &goast.Ident{Name: "string"}
	case ast.TypeInt:
		return &goast.Ident{Name: "int"}
	case ast.TypeFloat:
		return &goast.Ident{Name: "float64"}
	case ast.TypeBool:
		return &goast.Ident{Name: "bool"}
	case ast.TypeVoid:
		return &goast.Ident{Name: "void"}
	// Special case for now
	case "UUID":
		return &goast.Ident{Name: "string"}
	}
	return goast.NewIdent(string(ident))
}

func (t *Transformer) getAssertionBaseTypeIdent(assertion *ast.AssertionNode) (*goast.Ident, error) {
	if assertion.BaseType != nil {
		return transformTypeIdent(*assertion.BaseType), nil
	}
	typeNode, err := t.TypeChecker.LookupAssertionType(assertion, t.currentScope)
	if err != nil {
		err = fmt.Errorf("failed to lookup assertion type during getAssertionBaseTypeIdent: %w", err)
		log.WithError(err).Error("transforming assertion base type ident failed")
		return nil, err
	}
	return transformTypeIdent(typeNode.Ident), nil
}

func (t *Transformer) transformTypeDefExpr(expr ast.TypeDefExpr) (*goast.Expr, error) {
	switch e := expr.(type) {
	case ast.TypeDefAssertionExpr:
		baseTypeIdent, err := t.getAssertionBaseTypeIdent(e.Assertion)
		if err != nil {
			err = fmt.Errorf("failed to get assertion base type ident during transformation: %w", err)
			log.WithError(err).Error("transforming assertion base type ident failed")
			return nil, err
		}
		if baseTypeIdent.Name == "trpc.Mutation" || baseTypeIdent.Name == "trpc.Query" {
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
						expr, err := t.transformShapeType(shape)
						if err != nil {
							err = fmt.Errorf("failed to transform shape type during transformation: %w", err)
							log.WithError(err).Error("transforming shape type failed")
							return nil, err
						}
						inputField := goast.Field{
							Names: []*goast.Ident{goast.NewIdent("input")},
							Type:  *expr,
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
			return &expr, nil
		}

		var result goast.Expr = baseTypeIdent
		return &result, nil
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
		return &result, nil
	default:
		err := fmt.Errorf("unknown type def expr: %T", expr)
		log.WithError(err).Error("transforming type def expr failed")
		return nil, err
	}
}
