package transformer_go

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"

	log "github.com/sirupsen/logrus"
)

// TODO: Refactor to use a clear type model instead of direct AST transformation
// This will make it easier to:
// 1. Handle binary type expressions
// 2. Generate validation code
// 3. Support more type constraints
// 4. Improve type inference
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
		Doc: &goast.CommentGroup{
			List: []*goast.Comment{
				{
					Text: fmt.Sprintf("// %s", node.Expr.String()),
				},
			},
		},
	}, nil
}

func (t *Transformer) transformAssertionType(assertion *ast.AssertionNode) (*goast.Expr, error) {
	log.Tracef("transformAssertionType, assertion: %s", *assertion)
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
		lookupType, err := t.TypeChecker.LookupInferredType(field.Shape, true)
		if err != nil {
			err = fmt.Errorf("failed to lookup type during transformation: %w (key: %s)", err, t.TypeChecker.Hasher.HashNode(field.Shape).ToTypeIdent())
			log.WithError(err).Error("transforming type failed")
			return nil, err
		}
		log.Trace(fmt.Sprintf("transformShapeFieldType, lookupType: %s", lookupType[0]))
		shapeType := lookupType[0]
		result := transformTypeIdent(shapeType.Ident)
		var expr goast.Expr = result
		return &expr, nil
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

// TODO: Implement binary type expressions
// This should handle both conjunction (&) and disjunction (|) operators
// and generate appropriate validation code
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
	case *ast.TypeDefAssertionExpr:
		// Handle pointer by dereferencing and reusing value logic
		return t.transformTypeDefExpr(*e)
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

// TODO: Improve shape type registration
// This should handle nested shapes better and generate appropriate validation code
func (t *Transformer) defineShapeType(shape *ast.ShapeNode) error {
	// First register all nested shape fields
	if err := t.defineShapeFields(shape); err != nil {
		return err
	}

	// Then register the shape itself
	typeIdent := t.TypeChecker.Hasher.HashNode(shape).ToTypeIdent()

	// Create struct type for the shape
	structType, err := t.transformShapeType(shape)
	if err != nil {
		return fmt.Errorf("failed to transform shape type %s: %w", typeIdent, err)
	}

	decl, err := t.transformTypeDef(ast.TypeDefNode{
		Ident: typeIdent,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: nil,
				Constraints: []ast.ConstraintNode{{
					Name: "Shape",
					Args: []ast.ConstraintArgumentNode{{
						Shape: shape,
					}},
				}},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to transform shape type %s: %w", typeIdent, err)
	}

	// Override the type with our struct type
	decl.Specs[0].(*goast.TypeSpec).Type = *structType
	t.Output.AddType(decl)
	return nil
}

// defineShapeFields recursively registers type definitions for all shape fields
func (t *Transformer) defineShapeFields(shape *ast.ShapeNode) error {
	for _, field := range shape.Fields {
		if field.Shape != nil {
			// Recursively register nested shapes
			if err := t.defineShapeFields(field.Shape); err != nil {
				return err
			}

			// Register the shape field type
			typeIdent := t.TypeChecker.Hasher.HashNode(field.Shape).ToTypeIdent()

			// Create struct type for the shape field
			structType, err := t.transformShapeType(field.Shape)
			if err != nil {
				return fmt.Errorf("failed to transform shape field type %s: %w", typeIdent, err)
			}

			decl, err := t.transformTypeDef(ast.TypeDefNode{
				Ident: typeIdent,
				Expr: ast.TypeDefAssertionExpr{
					Assertion: &ast.AssertionNode{
						BaseType: nil,
						Constraints: []ast.ConstraintNode{{
							Name: "Shape",
							Args: []ast.ConstraintArgumentNode{{
								Shape: field.Shape,
							}},
						}},
					},
				},
			})
			if err != nil {
				return fmt.Errorf("failed to transform shape field type %s: %w", typeIdent, err)
			}

			// Override the type with our struct type
			decl.Specs[0].(*goast.TypeSpec).Type = *structType
			t.Output.AddType(decl)
		}
	}
	return nil
}

// defineShapeTypes finds all shapes in type definitions and registers them
func (t *Transformer) defineShapeTypes() error {
	for _, def := range t.TypeChecker.Defs {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if assertionExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
				if assertionExpr.Assertion != nil {
					for _, constraint := range assertionExpr.Assertion.Constraints {
						if len(constraint.Args) > 0 && constraint.Args[0].Shape != nil {
							if err := t.defineShapeType(constraint.Args[0].Shape); err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}
	return nil
}
