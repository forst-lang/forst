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
			baseType := field.Type.TypeParams[0]
			baseTypeField := ast.ShapeFieldNode{Type: &baseType}
			baseTypeExpr, err := t.transformShapeFieldType(baseTypeField)
			if err != nil {
				return nil, fmt.Errorf("failed to transform pointer base type: %w", err)
			}
			starExpr := &goast.StarExpr{X: *baseTypeExpr}
			var expr goast.Expr = starExpr
			return &expr, nil
		}

		// Always use the unified aliasing logic for all type names
		name, err := t.TypeChecker.GetAliasedTypeName(*field.Type)
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
		name, err := t.TypeChecker.GetAliasedTypeName(shapeType)
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
func (t *Transformer) findExistingTypeForShape(shape *ast.ShapeNode, expectedType *ast.TypeNode) (ast.TypeIdent, bool) {
	t.log.WithFields(logrus.Fields{
		"function":     "findExistingTypeForShape",
		"shape":        fmt.Sprintf("%+v", shape),
		"fields":       fmt.Sprintf("%+v", shape.Fields),
		"baseType":     shape.BaseType,
		"expectedType": expectedType,
	}).Debug("=== SHAPE MATCHING DEBUG ===")

	// If the shape has a BaseType, use it directly
	if shape.BaseType != nil {
		t.log.WithFields(logrus.Fields{
			"function": "findExistingTypeForShape",
			"baseType": *shape.BaseType,
		}).Debug("Using BaseType for shape literal")
		return *shape.BaseType, true
	}

	// If an expected type is provided, validate it's compatible with the shape
	if expectedType != nil {
		expectedTypeIdent := expectedType.Ident
		if def, exists := t.TypeChecker.Defs[expectedTypeIdent]; exists {
			// Check if the expected type definition is compatible with the shape
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
					t.log.WithFields(logrus.Fields{
						"function":      "findExistingTypeForShape",
						"expectedType":  expectedTypeIdent,
						"typeDefShape":  fmt.Sprintf("%+v", shapeExpr.Shape),
						"typeDefFields": fmt.Sprintf("%+v", shapeExpr.Shape.Fields),
						"literalShape":  fmt.Sprintf("%+v", shape),
						"literalFields": fmt.Sprintf("%+v", shape.Fields),
					}).Debug("Validating expected type compatibility")

					// Use the typechecker's ValidateShapeFields to check compatibility
					err := t.TypeChecker.ValidateShapeFields(*shape, shapeExpr.Shape.Fields, expectedTypeIdent)
					if err == nil {
						t.log.WithFields(logrus.Fields{
							"function":     "findExistingTypeForShape",
							"expectedType": expectedTypeIdent,
						}).Debug("Expected type is compatible, using it directly")
						return expectedTypeIdent, true
					} else {
						t.log.WithFields(logrus.Fields{
							"function":     "findExistingTypeForShape",
							"expectedType": expectedTypeIdent,
							"error":        err.Error(),
						}).Debug("Expected type is not compatible with shape, falling back to structural matching")
						// Expected type exists but doesn't match the shape structure
						// Fall back to structural matching instead of using incompatible type
					}
				}
			}
		} else {
			t.log.WithFields(logrus.Fields{
				"function":     "findExistingTypeForShape",
				"expectedType": expectedTypeIdent,
			}).Debug("Expected type not found in typechecker definitions")
		}
	}

	// Fall back to structural matching if no expected type or expected type doesn't match
	for typeIdent, def := range t.TypeChecker.Defs {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
				t.log.WithFields(logrus.Fields{
					"function":      "findExistingTypeForShape",
					"typeIdent":     typeIdent,
					"typeDefShape":  fmt.Sprintf("%+v", shapeExpr.Shape),
					"typeDefFields": fmt.Sprintf("%+v", shapeExpr.Shape.Fields),
					"literalShape":  fmt.Sprintf("%+v", shape),
					"literalFields": fmt.Sprintf("%+v", shape.Fields),
				}).Debug("Checking type definition for match")

				// Compare the shapes to see if they match
				if t.shapesMatch(shape, &shapeExpr.Shape) {
					t.log.WithFields(logrus.Fields{
						"function":  "findExistingTypeForShape",
						"typeIdent": typeIdent,
					}).Debug("Found matching type definition")
					return typeIdent, true
				} else {
					t.log.WithFields(logrus.Fields{
						"function":  "findExistingTypeForShape",
						"typeIdent": typeIdent,
					}).Debug("Type definition did not match")
				}
			} else {
				t.log.WithFields(logrus.Fields{
					"function":  "findExistingTypeForShape",
					"typeIdent": typeIdent,
					"defType":   fmt.Sprintf("%T", typeDef.Expr),
				}).Debug("Type definition is not a shape expression")
			}
		} else {
			t.log.WithFields(logrus.Fields{
				"function": "findExistingTypeForShape",
				"defType":  fmt.Sprintf("%T", def),
			}).Debug("Definition is not a TypeDefNode")
		}
	}
	t.log.WithFields(logrus.Fields{
		"function": "findExistingTypeForShape",
	}).Debug("No matching type definition found")
	return "", false
}

// shapesMatch compares two shapes to see if they have the same structure
func (t *Transformer) shapesMatch(shape1, shape2 *ast.ShapeNode) bool {
	t.log.WithFields(logrus.Fields{
		"function": "shapesMatch",
		"shape1":   fmt.Sprintf("%+v", shape1),
		"shape2":   fmt.Sprintf("%+v", shape2),
	}).Debug("=== SHAPE COMPARISON DEBUG ===")

	// Check if both shapes have the same number of fields
	if len(shape1.Fields) != len(shape2.Fields) {
		t.log.WithFields(logrus.Fields{
			"function": "shapesMatch",
			"len1":     len(shape1.Fields),
			"len2":     len(shape2.Fields),
		}).Debug("Field count mismatch")
		return false
	}

	// Compare each field
	for fieldName, field1 := range shape1.Fields {
		field2, exists := shape2.Fields[fieldName]
		if !exists {
			t.log.WithFields(logrus.Fields{
				"function":  "shapesMatch",
				"fieldName": fieldName,
			}).Debug("Field name not found in second shape")
			return false
		}

		// Debug: print type info for both fields
		t.log.WithFields(logrus.Fields{
			"function":        "shapesMatch",
			"fieldName":       fieldName,
			"field1Type":      fmt.Sprintf("%#v", field1.Type),
			"field2Type":      fmt.Sprintf("%#v", field2.Type),
			"field1Shape":     field1.Shape != nil,
			"field2Shape":     field2.Shape != nil,
			"field1Assertion": field1.Assertion != nil,
			"field2Assertion": field2.Assertion != nil,
		}).Debug("Comparing field types and shapes")

		if field1.Type != nil && field2.Type != nil {
			if field1.Type.Ident != field2.Type.Ident {
				t.log.WithFields(logrus.Fields{
					"function":   "shapesMatch",
					"fieldName":  fieldName,
					"field1Type": field1.Type.Ident,
					"field2Type": field2.Type.Ident,
				}).Debug("Field type mismatch")
				return false
			}
			// Optionally, compare pointer base types
			if field1.Type.Ident == ast.TypePointer && field2.Type.Ident == ast.TypePointer {
				if len(field1.Type.TypeParams) > 0 && len(field2.Type.TypeParams) > 0 {
					if field1.Type.TypeParams[0].Ident != field2.Type.TypeParams[0].Ident {
						t.log.WithFields(logrus.Fields{
							"function":      "shapesMatch",
							"fieldName":     fieldName,
							"field1PtrBase": field1.Type.TypeParams[0].Ident,
							"field2PtrBase": field2.Type.TypeParams[0].Ident,
						}).Debug("Pointer base type mismatch")
						return false
					}
				}
			}
		}
		// If only one side has a type, treat as compatible (literal omits explicit type)

		// For shape matching, we need to resolve the named type's field to its underlying shape
		// before comparing to the literal's shape. This handles pointers, aliases, and nested shapes.
		if field1.Shape != nil {
			// The literal provides a shape. Find the underlying shape of the named type's field.
			underlyingShape := t.resolveFieldToShape(field2)
			if underlyingShape != nil {
				if !t.shapesMatch(field1.Shape, underlyingShape) {
					t.log.WithFields(logrus.Fields{
						"function":  "shapesMatch",
						"fieldName": fieldName,
					}).Debug("Nested shape mismatch after resolving underlying shape")
					return false
				}
			} else {
				// Could not resolve underlying shape, treat as mismatch
				t.log.WithFields(logrus.Fields{
					"function":  "shapesMatch",
					"fieldName": fieldName,
				}).Debug("Could not resolve underlying shape for field")
				return false
			}
		} else if field2.Shape != nil {
			// The named type provides a shape, but the literal doesn't - mismatch
			t.log.WithFields(logrus.Fields{
				"function":  "shapesMatch",
				"fieldName": fieldName,
			}).Debug("Named type has shape but literal doesn't")
			return false
		}
		// If neither has a shape, we consider them compatible for matching purposes
	}

	t.log.WithFields(logrus.Fields{
		"function": "shapesMatch",
	}).Debug("Shapes match successfully")
	return true
}

// shapesCompatibleForExpectedType checks if two shapes are compatible for an expected type.
// This is a more lenient comparison that allows for different field names or types
// as long as the overall structure is compatible.
func (t *Transformer) shapesCompatibleForExpectedType(shape1, shape2 *ast.ShapeNode) bool {
	t.log.WithFields(logrus.Fields{
		"function": "shapesCompatibleForExpectedType",
		"shape1":   fmt.Sprintf("%+v", shape1),
		"shape2":   fmt.Sprintf("%+v", shape2),
	}).Debug("=== SHAPE COMPATIBILITY DEBUG ===")

	// Check if both shapes have the same number of fields
	if len(shape1.Fields) != len(shape2.Fields) {
		t.log.WithFields(logrus.Fields{
			"function": "shapesCompatibleForExpectedType",
			"len1":     len(shape1.Fields),
			"len2":     len(shape2.Fields),
		}).Debug("Field count mismatch")
		return false
	}

	// Compare each field
	for fieldName, field1 := range shape1.Fields {
		field2, exists := shape2.Fields[fieldName]
		if !exists {
			t.log.WithFields(logrus.Fields{
				"function":  "shapesCompatibleForExpectedType",
				"fieldName": fieldName,
			}).Debug("Field name not found in second shape")
			return false
		}

		// Debug: print type info for both fields
		t.log.WithFields(logrus.Fields{
			"function":        "shapesCompatibleForExpectedType",
			"fieldName":       fieldName,
			"field1Type":      fmt.Sprintf("%#v", field1.Type),
			"field2Type":      fmt.Sprintf("%#v", field2.Type),
			"field1Shape":     field1.Shape != nil,
			"field2Shape":     field2.Shape != nil,
			"field1Assertion": field1.Assertion != nil,
			"field2Assertion": field2.Assertion != nil,
		}).Debug("Comparing field types and shapes for compatibility")

		// For compatibility, we only need to ensure that the types are compatible
		// or that one side has a shape and the other doesn't.
		// If both have types, they must be the same.
		if field1.Type != nil && field2.Type != nil {
			if field1.Type.Ident != field2.Type.Ident {
				t.log.WithFields(logrus.Fields{
					"function":   "shapesCompatibleForExpectedType",
					"fieldName":  fieldName,
					"field1Type": field1.Type.Ident,
					"field2Type": field2.Type.Ident,
				}).Debug("Field type mismatch for compatibility")
				return false
			}
			// Optionally, compare pointer base types
			if field1.Type.Ident == ast.TypePointer && field2.Type.Ident == ast.TypePointer {
				if len(field1.Type.TypeParams) > 0 && len(field2.Type.TypeParams) > 0 {
					if field1.Type.TypeParams[0].Ident != field2.Type.TypeParams[0].Ident {
						t.log.WithFields(logrus.Fields{
							"function":      "shapesCompatibleForExpectedType",
							"fieldName":     fieldName,
							"field1PtrBase": field1.Type.TypeParams[0].Ident,
							"field2PtrBase": field2.Type.TypeParams[0].Ident,
						}).Debug("Pointer base type mismatch for compatibility")
						return false
					}
				}
			}
		}
		// If only one side has a type, treat as compatible (literal omits explicit type)

		// For shape matching, we need to resolve the named type's field to its underlying shape
		// before comparing to the literal's shape. This handles pointers, aliases, and nested shapes.
		if field1.Shape != nil {
			// The literal provides a shape. Find the underlying shape of the named type's field.
			underlyingShape := t.resolveFieldToShape(field2)
			if underlyingShape != nil {
				// If the named type's field has a shape, and the literal doesn't,
				// or if the types are different, it's not compatible.
				if field2.Shape != nil || field1.Type != nil && field2.Type != nil && field1.Type.Ident != field2.Type.Ident {
					t.log.WithFields(logrus.Fields{
						"function":  "shapesCompatibleForExpectedType",
						"fieldName": fieldName,
					}).Debug("Named type has shape or type mismatch for compatibility")
					return false
				}
			} else {
				// Could not resolve underlying shape, treat as mismatch
				t.log.WithFields(logrus.Fields{
					"function":  "shapesCompatibleForExpectedType",
					"fieldName": fieldName,
				}).Debug("Could not resolve underlying shape for field for compatibility")
				return false
			}
		} else if field2.Shape != nil {
			// The named type provides a shape, but the literal doesn't - mismatch
			t.log.WithFields(logrus.Fields{
				"function":  "shapesCompatibleForExpectedType",
				"fieldName": fieldName,
			}).Debug("Named type has shape but literal doesn't for compatibility")
			return false
		}
		// If neither has a shape, we consider them compatible for matching purposes
	}

	t.log.WithFields(logrus.Fields{
		"function": "shapesCompatibleForExpectedType",
	}).Debug("Shapes are compatible for expected type")
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

// resolveFieldToShape resolves a field to its underlying shape by following type aliases and definitions.
func (t *Transformer) resolveFieldToShape(field ast.ShapeFieldNode) *ast.ShapeNode {
	// If the field already has a shape, return it
	if field.Shape != nil {
		return field.Shape
	}

	// If the field has a type, try to resolve it to a shape
	if field.Type != nil {
		// Check if the type is a hash-based type or named type that we can look up
		if def, exists := t.TypeChecker.Defs[field.Type.Ident]; exists {
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
					return &shapeExpr.Shape
				}
			}
		}
	}

	return nil
}
