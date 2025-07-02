package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"

	logrus "github.com/sirupsen/logrus"
)

func (t *Transformer) transformShapeFieldType(field ast.ShapeFieldNode) (*goast.Expr, error) {
	if field.Type != nil {
		t.log.WithFields(logrus.Fields{
			"function": "transformShapeFieldType",
			"type":     field.Type.Ident,
		}).Tracef("transformShapeFieldType, type: %s", field.Type.Ident)

		// For user-defined types, ensure the type definition is emitted
		if !isGoBuiltinType(string(field.Type.Ident)) && field.Type.Ident != ast.TypeString && field.Type.Ident != ast.TypeInt && field.Type.Ident != ast.TypeFloat && field.Type.Ident != ast.TypeBool && field.Type.Ident != ast.TypeVoid {
			// Find the type definition and emit it
			for _, def := range t.TypeChecker.Defs {
				if typeDef, ok := def.(ast.TypeDefNode); ok {
					if typeDef.Ident == field.Type.Ident {
						// Transform the type definition and add it to output
						decl, err := t.transformTypeDef(typeDef)
						if err != nil {
							err = fmt.Errorf("failed to transform referenced type definition during transformation: %w", err)
							t.log.WithFields(logrus.Fields{
								"function": "transformShapeFieldType",
								"type":     field.Type.Ident,
							}).WithError(err).Error("transforming referenced type definition failed")
							return nil, err
						}
						// Only add if not already present
						if !t.Output.HasType(decl.Specs[0].(*goast.TypeSpec).Name.Name) {
							t.Output.AddType(decl)
						}
						break
					}
				}
			}
		}

		// Handle pointer types specially
		if field.Type.Ident == ast.TypePointer {
			if len(field.Type.TypeParams) == 0 {
				return nil, fmt.Errorf("pointer type must have a base type parameter")
			}
			// Ensure the base type definition is emitted (especially for value assertions)
			baseType := field.Type.TypeParams[0]
			if baseType.Ident == ast.TypeAssertion && baseType.Assertion != nil {
				// Handle assertion types directly
				baseTypeExpr, err := t.transformAssertionType(baseType.Assertion)
				if err != nil {
					return nil, fmt.Errorf("failed to transform pointer base assertion type: %w", err)
				}
				starExpr := &goast.StarExpr{X: *baseTypeExpr}
				var expr goast.Expr = starExpr
				return &expr, nil
			} else {
				// Handle other base types
				baseTypeField := ast.ShapeFieldNode{Type: &baseType}
				baseTypeExpr, err := t.transformShapeFieldType(baseTypeField)
				if err != nil {
					return nil, fmt.Errorf("failed to transform pointer base type: %w", err)
				}
				starExpr := &goast.StarExpr{X: *baseTypeExpr}
				var expr goast.Expr = starExpr
				return &expr, nil
			}
		}

		name, err := t.getTypeAliasNameForTypeNode(*field.Type)
		if err != nil {
			err = fmt.Errorf("failed to get type alias name during transformation: %w", err)
			t.log.WithFields(logrus.Fields{
				"function": "transformShapeFieldType",
			}).WithError(err).Error("getting type alias name failed")
			return nil, err
		}
		ident := goast.NewIdent(name)
		var expr goast.Expr = ident
		return &expr, nil
	}

	if field.Assertion != nil {
		t.log.WithFields(logrus.Fields{
			"function":  "transformShapeFieldType",
			"assertion": fmt.Sprintf("%+v", *field.Assertion),
		}).Tracef("Transforming shape field with assertion")
		expr, err := t.transformAssertionType(field.Assertion)
		if err != nil {
			err = fmt.Errorf("failed to transform assertion type during transformation: %w", err)
			t.log.WithFields(logrus.Fields{
				"function": "transformShapeFieldType",
				"error":    err,
			}).WithError(err).Error("transforming assertion type failed")
			return nil, err
		}
		return expr, nil
	}

	if field.Shape != nil {
		t.log.WithFields(logrus.Fields{
			"function": "transformShapeFieldType",
			"shape":    fmt.Sprintf("%+v", *field.Shape),
		}).Tracef("Transforming shape field with shape")
		lookupType, err := t.TypeChecker.LookupInferredType(field.Shape, true)
		if err != nil {
			hash, err := t.TypeChecker.Hasher.HashNode(field.Shape)
			if err != nil {
				err = fmt.Errorf("failed to hash shape during transformation: %w", err)
				t.log.WithFields(logrus.Fields{
					"function": "transformShapeFieldType",
				}).WithError(err).Error("transforming shape failed")
				return nil, err
			}
			err = fmt.Errorf("failed to lookup type during transformation: %w (key: %s)", err, hash.ToTypeIdent())
			t.log.WithFields(logrus.Fields{
				"function": "transformShapeFieldType",
			}).WithError(err).Error("transforming type failed")
			return nil, err
		}
		t.log.WithFields(logrus.Fields{
			"function": "transformShapeFieldType",
			"shape":    fmt.Sprintf("%+v", lookupType[0]),
		}).Tracef("Found inferred type of field of shape")
		shapeType := lookupType[0]
		name, err := t.getTypeAliasNameForTypeNode(shapeType)
		if err != nil {
			err = fmt.Errorf("failed to get type alias name during transformation: %w", err)
			t.log.WithFields(logrus.Fields{
				"function": "transformShapeFieldType",
				"error":    err,
			}).WithError(err).Error("getting type alias name failed")
			return nil, err
		}
		ident := goast.NewIdent(name)
		var expr goast.Expr = ident
		return &expr, nil
	}
	return nil, fmt.Errorf("shape field has neither explicit type nor assertion nor shape: %T", field)
}

func (t *Transformer) transformShapeType(shape *ast.ShapeNode) (*goast.Expr, error) {
	fields := []*goast.Field{}
	for name, field := range shape.Fields {
		fieldType, err := t.transformShapeFieldType(field)
		if err != nil {
			err = fmt.Errorf("failed to transform shape field type during transformation: %w", err)
			t.log.WithFields(logrus.Fields{
				"function": "transformShapeType",
			}).WithError(err).Error("transforming shape field type failed")
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
