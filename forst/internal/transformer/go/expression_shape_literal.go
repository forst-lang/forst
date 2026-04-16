package transformergo

import (
	"fmt"
	"strings"

	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
	"go/token"

	logrus "github.com/sirupsen/logrus"
)

// transformAssertionValue transforms an assertion value to a Go expression
func (t *Transformer) transformAssertionValue(assertion *ast.AssertionNode, expectedType *ast.TypeNode) (goast.Expr, error) {
	// Check if this is a Value assertion with a value argument
	if len(assertion.Constraints) > 0 {
		constraint := assertion.Constraints[0]
		if constraint.Name == ast.ValueConstraint && len(constraint.Args) > 0 {
			arg := constraint.Args[0]
			if arg.Value != nil {
				// For Value assertions, we need to handle the value appropriately
				switch v := (*arg.Value).(type) {
				case ast.ReferenceNode:
					// For Value(Ref(x)), we want a pointer to x when used in a pointer context
					// So we transform the inner value and take its address
					innerExpr, err := t.transformExpression(v.Value)
					if err != nil {
						return nil, err
					}

					if expectedType != nil && expectedType.Ident == ast.TypePointer {
						// Always add & for ReferenceNode in pointer context
						// The outer context expects a pointer value
						return &goast.UnaryExpr{Op: token.AND, X: innerExpr}, nil
					}
					return innerExpr, nil
				default:
					// For other value types, transform normally
					return t.transformExpression(*arg.Value)
				}
			}
		}
	}

	// For other assertion types, return a zero value based on the expected type
	// This is a fallback - in practice, we should handle more assertion types
	return goast.NewIdent("nil"), nil
}

// determineStructType robustly enforce named type for struct literals
func (t *Transformer) determineStructType(shape *ast.ShapeNode, expectedType *ast.TypeNode) (goast.Expr, error) {
	t.log.WithFields(logrus.Fields{
		"function":     "determineStructType",
		"expectedType": expectedType,
		"expectedTypeKind": func() ast.TypeKind {
			if expectedType != nil {
				return expectedType.TypeKind
			}
			return ast.TypeKindHashBased
		}(),
		"isUserDefined": func() bool {
			if expectedType != nil {
				return expectedType.IsUserDefined()
			}
			return false
		}(),
	}).Debug("[PINPOINT] determineStructType entry")

	// If expectedType is provided and is a named type (not hash-based), use it directly
	if expectedType != nil && expectedType.TypeKind != ast.TypeKindHashBased {
		var typeName string

		// Handle pointer types properly
		if expectedType.Ident == ast.TypePointer && len(expectedType.TypeParams) > 0 {
			// For pointer types like *AppContext, extract the underlying type
			underlyingType := expectedType.TypeParams[0]
			if underlyingType.TypeKind != ast.TypeKindHashBased {
				typeName = "*" + string(underlyingType.Ident)
			} else {
				// Fall back to hash-based type if underlying type is also hash-based
				typeName = string(underlyingType.Ident)
			}
		} else if strings.HasPrefix(string(expectedType.Ident), "*") {
			// For pointer type identifiers like *User, use as-is
			typeName = string(expectedType.Ident)
		} else {
			// For regular named types, use as-is
			typeName = string(expectedType.Ident)
		}

		t.log.WithFields(logrus.Fields{
			"function": "determineStructType",
			"typeName": typeName,
			"result":   "using expected type",
		}).Debug("[PINPOINT] determineStructType result")

		return goast.NewIdent(typeName), nil
	}

	// If expectedType is provided but is hash-based, use it directly
	if expectedType != nil {
		return goast.NewIdent(string(expectedType.Ident)), nil
	}

	// When no expected type is provided, check if we should use a named type
	// instead of generating a hash-based type
	if typeIdent, found := t.findExistingTypeForShape(shape, nil); found {
		return goast.NewIdent(string(typeIdent)), nil
	}

	// Generate a hash-based type name
	hash, err := t.TypeChecker.Hasher.HashNode(*shape)
	if err != nil {
		return nil, fmt.Errorf("failed to hash shape: %v", err)
	}

	inferredType := ast.TypeNode{Ident: hash.ToTypeIdent()}
	// Use robust type selection to find the best named type
	bestType := t.findBestNamedTypeForStructLiteral(inferredType, expectedType)

	// Ensure the type is emitted
	if err := t.defineShapeType(shape); err != nil {
		return nil, err
	}

	return goast.NewIdent(string(bestType.Ident)), nil
}

// buildFieldsForExpectedType builds field expressions when an expected type is provided
func (t *Transformer) buildFieldsForExpectedType(shape *ast.ShapeNode, expectedType *ast.TypeNode) ([]*goast.KeyValueExpr, error) {
	t.log.WithFields(logrus.Fields{
		"function":     "buildFieldsForExpectedType",
		"expectedType": expectedType.Ident,
		"shapeFields":  len(shape.Fields),
	}).Debug("[PINPOINT] buildFieldsForExpectedType entry")

	fields := make([]*goast.KeyValueExpr, 0)

	def, ok := t.TypeChecker.Defs[expectedType.Ident]
	if !ok {
		t.log.WithFields(logrus.Fields{
			"function":     "buildFieldsForExpectedType",
			"expectedType": expectedType.Ident,
		}).Debug("[PINPOINT] No type definition found")
		return fields, nil
	}

	typeDef, ok := def.(ast.TypeDefNode)
	if !ok {
		t.log.WithFields(logrus.Fields{
			"function": "buildFieldsForExpectedType",
			"defType":  fmt.Sprintf("%T", def),
		}).Debug("[PINPOINT] Definition is not a TypeDefNode")
		return fields, nil
	}

	payload, ok := ast.PayloadShape(typeDef.Expr)
	if !ok {
		t.log.WithFields(logrus.Fields{
			"function": "buildFieldsForExpectedType",
			"exprType": fmt.Sprintf("%T", typeDef.Expr),
		}).Debug("[PINPOINT] TypeDef expression has no struct payload (shape or error)")
		return fields, nil
	}

	// Only emit fields that exist in the Go struct type, and fill missing ones with zero values
	for fieldName, fieldDef := range payload.Fields {
		t.log.WithFields(logrus.Fields{
			"function":     "buildFieldsForExpectedType",
			"fieldName":    fieldName,
			"fieldDefType": fieldDef.Type,
			"hasAssertion": fieldDef.Assertion != nil,
		}).Debug("[PINPOINT] Processing field")

		// Resolve assertion fields to their actual types
		resolvedFieldDef := fieldDef
		if fieldDef.Assertion != nil && fieldDef.Type == nil {
			// For assertion fields like ctx:AppContext, resolve the assertion to get the actual type
			if assertionType, err := t.TypeChecker.InferAssertionType(fieldDef.Assertion, false, fieldName, nil); err == nil && len(assertionType) > 0 {
				// If the assertion refers to a named type, use that as the type ident
				if fieldDef.Assertion.BaseType != nil && assertionType[0].TypeKind != ast.TypeKindHashBased {
					// Use the named type from the assertion's BaseType
					resolvedFieldDef.Type = &ast.TypeNode{Ident: *fieldDef.Assertion.BaseType}
				} else {
					resolvedFieldDef.Type = &assertionType[0]
				}
			}
		}

		var fieldValue goast.Expr
		var err error

		if field, ok := shape.Fields[fieldName]; ok {
			// Always use resolvedFieldDef.Type as the expected type for the field
			expectedTypeForField := resolvedFieldDef.Type
			// PINPOINT: About to build field value
			fieldValue, err = t.buildFieldValue(field, &resolvedFieldDef, expectedTypeForField)
			if err != nil {
				return nil, err
			}
		} else {
			// Field not found in shape, use default value
			fieldValue, err = t.buildFieldValue(ast.ShapeFieldNode{}, &resolvedFieldDef, resolvedFieldDef.Type)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to build field %s: %v", fieldName, err)
		}

		goFieldName := fieldName
		if t.ExportReturnStructFields {
			goFieldName = capitalizeFirst(fieldName)
		}

		fields = append(fields, &goast.KeyValueExpr{
			Key:   goast.NewIdent(goFieldName),
			Value: fieldValue,
		})
	}

	t.log.WithFields(logrus.Fields{
		"function":     "buildFieldsForExpectedType",
		"fieldCount":   len(fields),
		"expectedType": expectedType.Ident,
	}).Debug("[PINPOINT] buildFieldsForExpectedType result")

	return fields, nil
}

// buildFieldsForShape builds field expressions when no expected type is provided
func (t *Transformer) buildFieldsForShape(shape *ast.ShapeNode) ([]*goast.KeyValueExpr, error) {
	fields := make([]*goast.KeyValueExpr, 0)

	for fieldName, field := range shape.Fields {
		fieldValue, err := t.buildFieldValue(field, &field, nil)
		if err != nil {
			return nil, err
		}

		goFieldName := fieldName
		if t.ExportReturnStructFields {
			goFieldName = capitalizeFirst(fieldName)
		}

		fields = append(fields, &goast.KeyValueExpr{
			Key:   goast.NewIdent(goFieldName),
			Value: fieldValue,
		})
	}

	return fields, nil
}

// buildFieldValue robustly enforce named type for struct literals
func (t *Transformer) buildFieldValue(field ast.ShapeFieldNode, fieldDef *ast.ShapeFieldNode, expectedTypeForField *ast.TypeNode) (goast.Expr, error) {
	var value goast.Expr
	var err error

	if field.Node != nil {
		if shapeNode, ok := field.Node.(ast.ShapeNode); ok {
			var fieldExpectedType *ast.TypeNode
			// Always use the field's declared type if it is a named type
			if fieldDef.Type != nil && fieldDef.Type.TypeKind != ast.TypeKindHashBased {
				fieldExpectedType = fieldDef.Type
			} else if fieldDef.Type != nil {
				fieldExpectedType = fieldDef.Type
			} else {
				fieldExpectedType = expectedTypeForField
			}
			value, err = t.transformShapeNodeWithExpectedType(&shapeNode, fieldExpectedType)
		} else if vn, ok := field.Node.(ast.VariableNode); ok && expectedTypeForField != nil && expectedTypeForField.IsResultType() &&
			len(expectedTypeForField.TypeParams) >= 2 && expectedTypeForField.TypeParams[1].Ident == ast.TypeError &&
			t.resultLocalSplit != nil {
			if split, ok := t.resultLocalSplit[string(vn.Ident.ID)]; ok && split.errGoName != "" && len(split.successGoNames) >= 1 {
				st, err2 := t.transformResultAsStructFieldGoType(*expectedTypeForField)
				if err2 != nil {
					return nil, err2
				}
				var typ goast.Expr = st
				value = &goast.CompositeLit{
					Type: typ,
					Elts: []goast.Expr{
						&goast.KeyValueExpr{Key: goast.NewIdent(loweredResultValueFieldName), Value: goast.NewIdent(split.successGoNames[0])},
						&goast.KeyValueExpr{Key: goast.NewIdent(loweredResultErrFieldName), Value: goast.NewIdent(split.errGoName)},
					},
				}
			} else {
				value, err = t.transformExpression(field.Node.(ast.ExpressionNode))
			}
		} else {
			value, err = t.transformExpression(field.Node.(ast.ExpressionNode))
		}
	} else if field.Shape != nil {
		var fieldExpectedType *ast.TypeNode
		// Always use the field's declared type if it is a named type
		if fieldDef.Type != nil && fieldDef.Type.TypeKind != ast.TypeKindHashBased {
			fieldExpectedType = fieldDef.Type
		} else if fieldDef.Type != nil {
			fieldExpectedType = fieldDef.Type
		} else {
			fieldExpectedType = expectedTypeForField
		}
		return t.transformShapeNodeWithExpectedType(field.Shape, fieldExpectedType)
	} else if field.Assertion != nil {
		value, err = t.transformAssertionValue(field.Assertion, nil)
	} else if field.Type != nil {
		value, err = t.buildTypeValue(field.Type)
	} else {
		// For missing fields (error cases), generate proper zero values instead of nil
		if fieldDef.Type != nil {
			// Use the same logic as buildZeroCompositeLiteral for consistent zero value generation
			switch fieldDef.Type.Ident {
			case ast.TypeString:
				value = &goast.BasicLit{Kind: token.STRING, Value: "\"\""}
			case ast.TypeInt:
				value = &goast.BasicLit{Kind: token.INT, Value: "0"}
			case ast.TypeBool:
				value = goast.NewIdent("false")
			case ast.TypeFloat:
				value = &goast.BasicLit{Kind: token.FLOAT, Value: "0.0"}
			case ast.TypeError:
				value = goast.NewIdent("nil")
			default:
				// For user-defined types, use nil
				value = goast.NewIdent("nil")
			}
		} else {
			value = goast.NewIdent("nil")
		}
	}

	if err != nil {
		return nil, err
	}

	// After generating value, wrap in & for pointer fields if we have a value
	if fieldDef.Type != nil {
		isPointer := fieldDef.Type.Ident == ast.TypePointer ||
			(len(string(fieldDef.Type.Ident)) > 0 && string(fieldDef.Type.Ident)[0] == '*') ||
			(fieldDef.Type.Ident == "Pointer" && len(fieldDef.Type.TypeParams) > 0)

		if isPointer {
			// If we have a value (not nil), wrap it in &
			if value != nil {
				// Check if the value is already "nil" (which shouldn't be wrapped)
				if ident, ok := value.(*goast.Ident); ok && ident.Name == "nil" {
					return value, nil
				}

				// Avoid double address-of: if value is already &expr, don't wrap again
				if unary, ok := value.(*goast.UnaryExpr); ok && unary.Op == token.AND {
					return value, nil
				}

				if composite, ok := value.(*goast.CompositeLit); ok {
					// For pointer fields with struct literals, wrap in & but ensure the composite literal
					if ident, ok := composite.Type.(*goast.Ident); ok && strings.HasPrefix(ident.Name, "*") {
						baseTypeName := strings.TrimPrefix(ident.Name, "*")
						composite.Type = goast.NewIdent(baseTypeName)
					}
					return &goast.UnaryExpr{
						Op: token.AND,
						X:  composite,
					}, nil
				}
				// For other values (like variable references), wrap in &
				return &goast.UnaryExpr{
					Op: token.AND,
					X:  value,
				}, nil
			}
			// If no value provided, generate nil
			return goast.NewIdent("nil"), nil
		}
	}

	return value, nil
}

// buildTypeValue builds a value for a type field
func (t *Transformer) buildTypeValue(fieldType *ast.TypeNode) (goast.Expr, error) {
	// Handle pointer types (both Pointer(Type) and *Type formats)
	if fieldType.Ident == ast.TypePointer && len(fieldType.TypeParams) > 0 {
		return goast.NewIdent("nil"), nil
	}

	// Handle pointer type identifiers like *String, *User
	identStr := string(fieldType.Ident)
	if strings.HasPrefix(identStr, "*") {
		return goast.NewIdent("nil"), nil
	}

	aliasName, err := t.TypeChecker.GetAliasedTypeName(*fieldType, typechecker.GetAliasedTypeNameOptions{AllowStructuralAlias: true})
	if err != nil || aliasName == "" {
		return goast.NewIdent("nil"), nil
	}

	switch aliasName {
	case "int":
		return &goast.BasicLit{Kind: token.INT, Value: "0"}, nil
	case "string":
		return &goast.BasicLit{Kind: token.STRING, Value: "\"\""}, nil
	case "bool":
		return goast.NewIdent("false"), nil
	case "float64":
		return &goast.BasicLit{Kind: token.FLOAT, Value: "0.0"}, nil
	default:
		if strings.HasPrefix(aliasName, "T_") {
			return goast.NewIdent("nil"), nil
		}
		return &goast.CompositeLit{Type: goast.NewIdent(aliasName)}, nil
	}
}

// transformShapeNodeWithExpectedType generates a struct literal using the expected type if possible
func (t *Transformer) transformShapeNodeWithExpectedType(shape *ast.ShapeNode, expectedType *ast.TypeNode) (goast.Expr, error) {
	// PINPOINT: Log entry with detailed context
	var ident string
	var typeKind ast.TypeKind
	var isUserDefined, isHashBased, isGoBuiltin, hasPrefixT bool

	if expectedType != nil {
		ident = string(expectedType.Ident)
		typeKind = expectedType.TypeKind
		isUserDefined = expectedType.IsUserDefined()
		isHashBased = expectedType.IsHashBased()
		isGoBuiltin = expectedType.IsGoBuiltin()
		hasPrefixT = strings.HasPrefix(string(expectedType.Ident), "T_")
	}

	t.log.WithFields(logrus.Fields{
		"function":      "transformShapeNodeWithExpectedType",
		"expectedType":  expectedType,
		"isNil":         expectedType == nil,
		"ident":         ident,
		"typeKind":      typeKind,
		"isUserDefined": isUserDefined,
		"isHashBased":   isHashBased,
		"isGoBuiltin":   isGoBuiltin,
		"hasPrefixT":    hasPrefixT,
	}).Debug("[PINPOINT] transformShapeNodeWithExpectedType entry")

	// If expectedType is a named type (not hash-based), always use it for the composite literal
	if expectedType != nil && expectedType.TypeKind != ast.TypeKindHashBased {
		t.log.WithFields(logrus.Fields{
			"function": "transformShapeNodeWithExpectedType",
			"result":   "using expected type for composite literal",
		}).Debug("[PINPOINT] Using expected type for composite literal")

		structType, err := t.determineStructType(shape, expectedType)
		if err != nil {
			return nil, err
		}
		fields, err := t.buildFieldsForExpectedType(shape, expectedType)
		if err != nil {
			return nil, err
		}
		// Convert []*goast.KeyValueExpr to []goast.Expr
		fieldExprs := make([]goast.Expr, len(fields))
		for i, f := range fields {
			fieldExprs[i] = f
		}
		return &goast.CompositeLit{
			Type: structType,
			Elts: fieldExprs,
		}, nil
	}

	t.log.WithFields(logrus.Fields{
		"function": "transformShapeNodeWithExpectedType",
		"result":   "falling back to structural matching",
	}).Debug("[PINPOINT] Falling back to structural matching")

	// Determine the struct type to use for the composite literal
	structType, err := t.determineStructType(shape, expectedType)
	if err != nil {
		return nil, err
	}

	// After determining structType, if it is a hash-based type (T_*), ensure its type definition is emitted.
	if ident, ok := structType.(*goast.Ident); ok {
		if strings.HasPrefix(ident.Name, "T_") {
			if def, exists := t.TypeChecker.Defs[ast.TypeIdent(ident.Name)]; exists {
				processed := make(map[ast.TypeIdent]bool)
				if err := t.emitTypeAndReferencedTypes(ast.TypeIdent(ident.Name), def, processed); err != nil {
					return nil, err
				}
			}
		}
	}

	// Build the fields
	var fields []*goast.KeyValueExpr
	if expectedType != nil {
		fields, err = t.buildFieldsForExpectedType(shape, expectedType)
	} else {
		fields, err = t.buildFieldsForShape(shape)
	}

	if err != nil {
		return nil, err
	}

	// Convert []*goast.KeyValueExpr to []goast.Expr
	fieldExprs := make([]goast.Expr, len(fields))
	for i, f := range fields {
		fieldExprs[i] = f
	}
	return &goast.CompositeLit{
		Type: structType,
		Elts: fieldExprs,
	}, nil
}

// findBestNamedTypeForStructLiteral finds the best named type to use for a struct literal
// by checking if the inferred type is structurally compatible with any named types
func (t *Transformer) findBestNamedTypeForStructLiteral(inferredType ast.TypeNode, expectedType *ast.TypeNode) ast.TypeNode {
	// If we already have a named type as expected type, use it
	if expectedType != nil && expectedType.TypeKind != ast.TypeKindHashBased {
		return *expectedType
	}

	// If the inferred type is already a named type, use it
	if inferredType.TypeKind != ast.TypeKindHashBased {
		return inferredType
	}

	// Check all type definitions for structural compatibility
	for typeIdent := range t.TypeChecker.Defs {
		// Skip hash-based types
		if strings.HasPrefix(string(typeIdent), "T_") {
			continue
		}

		// Check if this named type is structurally compatible with the inferred type
		if t.TypeChecker.IsTypeCompatible(inferredType, ast.TypeNode{Ident: typeIdent}) {
			return ast.TypeNode{Ident: typeIdent}
		}
	}

	// No compatible named type found, return the original inferred type
	return inferredType
}
