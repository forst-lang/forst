package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"

	logrus "github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

// findExistingTypeForShape checks if a shape literal matches an existing type definition
func (t *Transformer) findExistingTypeForShape(shape *ast.ShapeNode) (ast.TypeIdent, bool) {
	for typeIdent, def := range t.TypeChecker.Defs {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
				// Compare the shapes to see if they match
				if t.shapesMatch(shape, &shapeExpr.Shape) {
					return typeIdent, true
				}
			}
		}
	}
	return "", false
}

// shapesMatch checks if two shapes have the same structure
func (t *Transformer) shapesMatch(shape1, shape2 *ast.ShapeNode) bool {
	if len(shape1.Fields) != len(shape2.Fields) {
		return false
	}

	for name, field1 := range shape1.Fields {
		field2, exists := shape2.Fields[name]
		if !exists {
			return false
		}

		// For now, just check if both fields have the same type or assertion
		// This is a simplified comparison - in a full implementation, you'd want to compare the actual types
		if (field1.Type != nil) != (field2.Type != nil) {
			return false
		}
		if (field1.Assertion != nil) != (field2.Assertion != nil) {
			return false
		}
		if (field1.Shape != nil) != (field2.Shape != nil) {
			return false
		}
	}
	return true
}

// capitalizeFirst returns the input string with the first Unicode letter uppercased (robust, modern)
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}

	// Use the cases package for proper Unicode case conversion
	caser := cases.Upper(language.Und)
	upper := caser.String(s)

	// If the string was already uppercase or didn't change, return as is
	if upper == s {
		return s
	}

	// For proper capitalization, we want only the first character uppercase
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}

	// Convert only the first character to uppercase using the cases package
	firstChar := caser.String(string(runes[0]))
	if len(firstChar) == 0 {
		return s
	}

	// Combine the uppercase first character with the rest of the string
	return firstChar + string(runes[1:])
}

func (t *Transformer) transformShapeType(shape *ast.ShapeNode) (*goast.Expr, error) {
	// Always generate a struct type for shape definitions
	// The existing type matching is only for shape literals, not type definitions
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
		goFieldName := name
		if t.ExportReturnStructFields {
			goFieldName = capitalizeFirst(name)
		}
		fields = append(fields, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(goFieldName)},
			Type:  *fieldType,
			Tag:   &goast.BasicLit{Kind: token.STRING, Value: "`json:\"" + name + "\"`"},
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
