package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"

	logrus "github.com/sirupsen/logrus"
)

// getAssertionStringForError returns a properly qualified assertion string for error messages
func (t *Transformer) getAssertionStringForError(assertion *ast.AssertionNode) string {
	// If BaseType is set, use it with the constraint name
	if assertion.BaseType != nil {
		return assertion.ToString(assertion.BaseType)
	}

	// Otherwise, try to get the inferred type from the typechecker
	if inferredType, err := t.TypeChecker.LookupInferredType(assertion, false); err == nil && len(inferredType) > 0 {
		// Use the most specific non-hash-based alias for error messages
		nonHash := t.TypeChecker.GetMostSpecificNonHashAlias(inferredType[0])
		return assertion.ToString(&nonHash.Ident)
	}

	// Fallback to the original string representation
	return assertion.String()
}

// getZeroValue returns the zero value for a Go type
func getZeroValue(goType goast.Expr) goast.Expr {
	switch t := goType.(type) {
	case *goast.Ident:
		switch t.Name {
		case "string":
			return &goast.BasicLit{Kind: token.STRING, Value: "\"\""}
		case "int", "int64", "int32", "int16", "int8", "uint", "uint64", "uint32", "uint16", "uint8", "uintptr", "float64", "float32", "complex64", "complex128", "rune", "byte":
			return &goast.BasicLit{Kind: token.INT, Value: "0"}
		case "bool":
			return &goast.Ident{Name: "false"}
		case "error":
			return &goast.Ident{Name: "nil"}
		default:
			// For struct types, return nil
			return &goast.Ident{Name: "nil"}
		}
	case *goast.StarExpr:
		// For pointer types, return nil
		return &goast.Ident{Name: "nil"}
	case *goast.ArrayType:
		// For array types, return nil
		return &goast.Ident{Name: "nil"}
	case *goast.InterfaceType:
		// For interface types, return nil
		return &goast.Ident{Name: "nil"}
	default:
		// For any other type, return nil
		return &goast.Ident{Name: "nil"}
	}
}

// zeroValueExprForASTType returns a Go expression for the zero value of a Forst type.
// Named struct types use a composite literal; builtins use getZeroValue after transformType.
func (t *Transformer) zeroValueExprForASTType(ty ast.TypeNode) (goast.Expr, error) {
	if def, ok := t.TypeChecker.Defs[ty.Ident].(ast.TypeDefNode); ok {
		if _, ok := ast.PayloadShape(def.Expr); ok {
			return t.buildZeroCompositeLiteral(&ty), nil
		}
	}
	if ty.TypeKind == ast.TypeKindUserDefined || ty.TypeKind == ast.TypeKindHashBased {
		return t.buildZeroCompositeLiteral(&ty), nil
	}
	goType, err := t.transformType(ty)
	if err != nil {
		return nil, err
	}
	return getZeroValue(goType), nil
}

// findBestNamedTypeForReturnType tries to find a named type that matches the given hash-based type
func (t *Transformer) findBestNamedTypeForReturnType(hashType ast.TypeNode) string {
	// If it's already a named type (not hash-based), return it
	if !hashType.IsHashBased() {
		return string(hashType.Ident)
	}

	// Look through all named types to find one that's compatible
	for typeIdent, def := range t.TypeChecker.Defs {
		if _, ok := def.(ast.TypeDefNode); ok {
			// Skip hash-based types
			if string(typeIdent)[:2] == "T_" {
				continue
			}

			// Check if this named type is compatible with the hash-based type
			if t.TypeChecker.IsTypeCompatible(hashType, ast.TypeNode{Ident: typeIdent}) {
				return string(typeIdent)
			}
		}
	}

	return ""
}

// findBestNamedTypeForReturnStructLiteral finds the best named type for a struct literal in a return statement.
// It always attempts to find a structurally identical named type, even if the expected type is not hash-based.
func (t *Transformer) findBestNamedTypeForReturnStructLiteral(shapeNode ast.ShapeNode, expectedType *ast.TypeNode) *ast.TypeNode {
	// Always try to find a structurally identical named type for the shape
	if namedType := t.TypeChecker.FindAnyStructurallyIdenticalNamedType(shapeNode); namedType != "" {
		return &ast.TypeNode{Ident: namedType}
	}

	// If no structurally identical named type found, use the expected type
	if expectedType != nil {
		return expectedType
	}

	return nil
}

// Helper: Build a composite literal of the expected type, mapping a variable to the first field, zero for others
func (t *Transformer) buildCompositeLiteralForReturn(expectedType *ast.TypeNode, valueVar string) goast.Expr {
	// PINPOINT: Log when this function is called
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function":     "buildCompositeLiteralForReturn",
			"expectedType": expectedType.Ident,
			"valueVar":     valueVar,
		}).Debug("[PINPOINT] buildCompositeLiteralForReturn called")
	}

	def, ok := t.TypeChecker.Defs[expectedType.Ident].(ast.TypeDefNode)
	if !ok {
		return goast.NewIdent(valueVar)
	}
	payload, ok := ast.PayloadShape(def.Expr)
	if !ok {
		return goast.NewIdent(valueVar)
	}
	fields := payload.Fields
	elts := make([]goast.Expr, 0, len(fields))
	used := false
	for fieldName, field := range fields {
		var val goast.Expr
		if !used {
			val = goast.NewIdent(valueVar)
			used = true
		} else {
			// Use the same logic as buildZeroCompositeLiteral for consistent zero value generation
			switch field.Type.Ident {
			case ast.TypeString:
				val = &goast.BasicLit{Kind: token.STRING, Value: "\"\""}
			case ast.TypeInt:
				val = &goast.BasicLit{Kind: token.INT, Value: "0"}
			case ast.TypeBool:
				val = goast.NewIdent("false")
			case ast.TypeFloat:
				val = &goast.BasicLit{Kind: token.FLOAT, Value: "0.0"}
			case ast.TypeError:
				val = goast.NewIdent("nil")
			default:
				// For user-defined types, use nil
				val = goast.NewIdent("nil")
			}

			// PINPOINT: Log what value is being generated for each field
			if t.log != nil {
				t.log.WithFields(logrus.Fields{
					"function":     "buildCompositeLiteralForReturn",
					"fieldName":    fieldName,
					"fieldType":    field.Type.Ident,
					"valueType":    fmt.Sprintf("%T", val),
					"valueDetails": fmt.Sprintf("%+v", val),
				}).Debug("[PINPOINT] Generated value for field in buildCompositeLiteralForReturn")
			}
		}
		goFieldName := fieldName
		if t.ExportReturnStructFields {
			goFieldName = capitalizeFirst(fieldName)
		}

		elts = append(elts, &goast.KeyValueExpr{
			Key:   goast.NewIdent(goFieldName),
			Value: val,
		})
	}
	return &goast.CompositeLit{Type: goast.NewIdent(string(expectedType.Ident)), Elts: elts}
}

// Helper: Build a zero composite literal of the expected type
func (t *Transformer) buildZeroCompositeLiteral(expectedType *ast.TypeNode) goast.Expr {
	// PINPOINT: Log when this function is called
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"function":     "buildZeroCompositeLiteral",
			"expectedType": expectedType.Ident,
			"isHashBased":  expectedType.IsHashBased(),
			"typeKind":     expectedType.TypeKind,
		}).Debug("[PINPOINT] buildZeroCompositeLiteral called")
	}

	def, ok := t.TypeChecker.Defs[expectedType.Ident].(ast.TypeDefNode)
	if !ok {
		// DEBUG: Log when type definition is not found
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"expectedType": expectedType.Ident,
				"found":        false,
			}).Debug("[DEBUG] Type definition not found")
		}
		return goast.NewIdent("nil")
	}
	payload, ok := ast.PayloadShape(def.Expr)
	if !ok {
		return goast.NewIdent("nil")
	}
	fields := payload.Fields
	elts := make([]goast.Expr, 0, len(fields))
	for fieldName, field := range fields {
		// DEBUG: Log field processing
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"fieldName":     fieldName,
				"fieldType":     field.Type.Ident,
				"fieldTypeKind": field.Type.TypeKind,
			}).Debug("[DEBUG] Processing field for zero value")
		}

		var val goast.Expr
		switch field.Type.Ident {
		case ast.TypeString:
			val = &goast.BasicLit{Kind: token.STRING, Value: "\"\""}
		case ast.TypeInt:
			val = &goast.BasicLit{Kind: token.INT, Value: "0"}
		case ast.TypeBool:
			val = goast.NewIdent("false")
		case ast.TypeFloat:
			val = &goast.BasicLit{Kind: token.FLOAT, Value: "0.0"}
		case ast.TypeError:
			val = goast.NewIdent("nil")
		default:
			// For user-defined types, recursively generate zero values
			if field.Type.TypeKind == ast.TypeKindUserDefined {
				// Recursively build zero value for the user-defined type
				val = t.buildZeroCompositeLiteral(field.Type)
			} else {
				// For other types, use nil
				val = goast.NewIdent("nil")
			}
		}

		// DEBUG: Log the generated zero value
		if t.log != nil {
			t.log.WithFields(logrus.Fields{
				"fieldName": fieldName,
				"fieldType": field.Type.Ident,
				"zeroValue": fmt.Sprintf("%T", val),
				"zeroValueDetails": func() string {
					if basicLit, ok := val.(*goast.BasicLit); ok {
						return fmt.Sprintf("BasicLit{Kind:%s, Value:%s}", basicLit.Kind, basicLit.Value)
					}
					if ident, ok := val.(*goast.Ident); ok {
						return fmt.Sprintf("Ident{Name:%s}", ident.Name)
					}
					return fmt.Sprintf("%T", val)
				}(),
			}).Debug("[DEBUG] Generated zero value for field")
		}

		goFieldName := fieldName
		if t.ExportReturnStructFields {
			goFieldName = capitalizeFirst(fieldName)
		}

		elts = append(elts, &goast.KeyValueExpr{
			Key:   goast.NewIdent(goFieldName),
			Value: val,
		})
	}

	// DEBUG: Log the final generated composite literal
	if t.log != nil {
		t.log.WithFields(logrus.Fields{
			"expectedType":  expectedType.Ident,
			"fieldCount":    len(fields),
			"generatedType": string(expectedType.Ident),
		}).Debug("[DEBUG] Generated zero composite literal")
	}

	return &goast.CompositeLit{Type: goast.NewIdent(string(expectedType.Ident)), Elts: elts}
}

// wrapVariableInNamedStruct generates a Go composite literal of the expected named struct type, mapping the variable to the appropriate field.
func (t *Transformer) wrapVariableInNamedStruct(expectedType *ast.TypeNode, variable ast.VariableNode) (goast.Expr, error) {
	goType, _ := t.transformType(*expectedType)

	// Get the struct definition to understand its fields
	def, ok := t.TypeChecker.Defs[expectedType.Ident].(ast.TypeDefNode)
	if !ok {
		return nil, fmt.Errorf("wrapVariableInNamedStruct: expected type %v is not a TypeDefNode", expectedType.Ident)
	}

	payload, ok := ast.PayloadShape(def.Expr)
	if !ok {
		return nil, fmt.Errorf("wrapVariableInNamedStruct: expected type %v does not have a shape or error payload", expectedType.Ident)
	}

	// Create composite literal with all fields
	elts := make([]goast.Expr, 0, len(payload.Fields))

	for fieldName := range payload.Fields {
		var value goast.Expr

		// Check if this field should contain our variable
		fieldType := payload.Fields[fieldName].Type
		if fieldType != nil && fieldType.Ident == ast.TypeIdent(variable.Ident.ID) {
			// This field should contain our variable
			value = goast.NewIdent(string(variable.Ident.ID))
		} else {
			// This field needs a default value
			// Use proper zero values based on field type
			fieldType := payload.Fields[fieldName].Type
			if fieldType != nil {
				switch fieldType.Ident {
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
				// Fallback to 0 if field type is unknown
				value = goast.NewIdent("0")
			}
		}

		goFieldName := fieldName
		if t.ExportReturnStructFields {
			goFieldName = capitalizeFirst(fieldName)
		}

		elts = append(elts, &goast.KeyValueExpr{
			Key:   goast.NewIdent(goFieldName),
			Value: value,
		})
	}

	return &goast.CompositeLit{
		Type: goType,
		Elts: elts,
	}, nil
}

// getShapeNode extracts a *ast.ShapeNode from an ast.Node, handling both value and pointer types
func getShapeNode(value ast.Node) (*ast.ShapeNode, bool) {
	if sn, ok := value.(*ast.ShapeNode); ok {
		return sn, true
	}
	if snVal, ok := value.(ast.ShapeNode); ok {
		return &snVal, true
	}
	return nil, false
}
