package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// inferValueConstraintType attempts to infer the type from a Value constraint
func (tc *TypeChecker) inferValueConstraintType(constraint ast.ConstraintNode, fieldName string, expectedType *ast.TypeNode) (ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"function":   "inferValueConstraintType",
		"fieldName":  fieldName,
		"constraint": fmt.Sprintf("%+v", constraint),
	}).Debugf("Starting Value constraint type inference")

	if len(constraint.Args) == 0 {
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
		}).Debugf("No arguments in Value constraint")
		return ast.TypeNode{}, fmt.Errorf("could not infer type for Value constraint on field '%s'", fieldName)
	}

	arg := constraint.Args[0]
	if arg.Value == nil {
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
			"argValue":  nil,
		}).Debugf("Value argument is nil before type switch")
		return ast.TypeNode{}, fmt.Errorf("nil value in Value constraint")
	}
	// Debug: print type of arg.Value and dereferenced value
	tc.log.WithFields(logrus.Fields{
		"function":     "inferValueConstraintType",
		"fieldName":    fieldName,
		"argValueType": fmt.Sprintf("%T", arg.Value),
		"argValuePtr":  fmt.Sprintf("%p", arg.Value),
	}).Debugf("Type of arg.Value before dereference")
	value := *arg.Value // ValueNode interface
	tc.log.WithFields(logrus.Fields{
		"function":          "inferValueConstraintType",
		"fieldName":         fieldName,
		"dereferencedType":  fmt.Sprintf("%T", value),
		"dereferencedValue": fmt.Sprintf("%+v", value),
	}).Debugf("Type and value after dereferencing arg.Value")
	switch v := value.(type) {
	case *ast.VariableNode:
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
			"varIdent":  v.Ident.ID,
		}).Debugf("Processing VariableNode in Value constraint")
		// Look up the variable's actual type
		varType, err := tc.LookupVariableType(v, tc.CurrentScope())
		if err == nil {
			tc.log.WithFields(logrus.Fields{
				"function":  "inferValueConstraintType",
				"fieldName": fieldName,
				"varType":   varType.Ident,
			}).Debugf("Successfully inferred type from variable in Value constraint")
			return varType, nil
		}

		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
			"variable":  v.Ident.ID,
			"error":     err.Error(),
		}).Debugf("Failed to lookup variable type, trying dot notation")

		// If variable type lookup fails, try to infer from the variable name
		// For dot notation like "query.id", try to infer from context
		if strings.Contains(string(v.Ident.ID), ".") {
			parts := strings.Split(string(v.Ident.ID), ".")
			tc.log.WithFields(logrus.Fields{
				"function":  "inferValueConstraintType",
				"fieldName": fieldName,
				"variable":  v.Ident.ID,
				"parts":     parts,
			}).Debugf("Processing dot notation variable")

			if len(parts) >= 2 {
				// Try to look up the type of the base variable
				baseVar := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(parts[0])}}
				tc.log.WithFields(logrus.Fields{
					"function":  "inferValueConstraintType",
					"fieldName": fieldName,
					"baseVar":   parts[0],
				}).Debugf("Looking up base variable type")

				baseType, err := tc.LookupVariableType(&baseVar, tc.CurrentScope())
				if err == nil {
					tc.log.WithFields(logrus.Fields{
						"function":  "inferValueConstraintType",
						"fieldName": fieldName,
						"baseType":  baseType.Ident,
					}).Debugf("Found base variable type, looking up field")

					// Now try to look up the field on the base type
					fieldType, err := tc.lookupFieldPath(baseType, parts[1:])
					if err == nil {
						tc.log.WithFields(logrus.Fields{
							"function":  "inferValueConstraintType",
							"fieldName": fieldName,
							"fieldType": fieldType.Ident,
						}).Debugf("Successfully inferred type from field access in Value constraint")
						return fieldType, nil
					}
					tc.log.WithFields(logrus.Fields{
						"function":  "inferValueConstraintType",
						"fieldName": fieldName,
						"baseType":  baseType.Ident,
						"fieldPath": parts[1:],
						"error":     err.Error(),
					}).Debugf("Failed to lookup field on base type")
				} else {
					tc.log.WithFields(logrus.Fields{
						"function":  "inferValueConstraintType",
						"fieldName": fieldName,
						"baseVar":   parts[0],
						"error":     err.Error(),
					}).Debugf("Failed to lookup base variable type")
				}
			}
		}

		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
			"variable":  v.Ident.ID,
		}).Debugf("All attempts to infer type failed")
		return ast.TypeNode{}, fmt.Errorf("could not infer type for Value constraint on field '%s'", fieldName)

	case ast.VariableNode:
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
			"varIdent":  v.Ident.ID,
		}).Debugf("Processing VariableNode (non-pointer) in Value constraint")
		// Take address and reuse pointer logic inline
		vPtr := &v
		varType, err := tc.LookupVariableType(vPtr, tc.CurrentScope())
		if err == nil {
			tc.log.WithFields(logrus.Fields{
				"function":  "inferValueConstraintType",
				"fieldName": fieldName,
				"varType":   varType.Ident,
			}).Debugf("Successfully inferred type from variable in Value constraint (non-pointer)")
			return varType, nil
		}
		// If variable type lookup fails, try to infer from the variable name
		// For dot notation like "query.id", try to infer from context
		if strings.Contains(string(v.Ident.ID), ".") {
			parts := strings.Split(string(v.Ident.ID), ".")
			tc.log.WithFields(logrus.Fields{
				"function":  "inferValueConstraintType",
				"fieldName": fieldName,
				"variable":  v.Ident.ID,
				"parts":     parts,
			}).Debugf("Processing dot notation variable (non-pointer)")

			if len(parts) >= 2 {
				baseVar := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(parts[0])}}
				baseType, err := tc.LookupVariableType(&baseVar, tc.CurrentScope())
				if err == nil {
					fieldType, err := tc.lookupFieldPath(baseType, parts[1:])
					if err == nil {
						tc.log.WithFields(logrus.Fields{
							"function":  "inferValueConstraintType",
							"fieldName": fieldName,
							"fieldType": fieldType.Ident,
						}).Debugf("Successfully inferred type from field access in Value constraint (non-pointer)")
						return fieldType, nil
					}
				}
			}
		}
		// Fallback: use expectedType if provided
		if expectedType != nil {
			tc.log.WithFields(logrus.Fields{
				"function":     "inferValueConstraintType",
				"fieldName":    fieldName,
				"expectedType": expectedType.Ident,
			}).Debugf("Falling back to expectedType for Value constraint (non-pointer)")
			return *expectedType, nil
		}
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
			"variable":  v.Ident.ID,
		}).Debugf("All attempts to infer type failed (non-pointer)")
		return ast.TypeNode{}, fmt.Errorf("could not infer type for Value constraint on field '%s'", fieldName)
	case *ast.StringLiteralNode:
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
		}).Debugf("Processing StringLiteralNode in Value constraint")
		return ast.TypeNode{Ident: ast.TypeString}, nil
	case ast.StringLiteralNode:
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
		}).Debugf("Processing StringLiteralNode (non-pointer) in Value constraint")
		return ast.TypeNode{Ident: ast.TypeString}, nil
	case *ast.IntLiteralNode:
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
		}).Debugf("Processing IntLiteralNode in Value constraint")
		return ast.TypeNode{Ident: ast.TypeInt}, nil
	case ast.IntLiteralNode:
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
		}).Debugf("Processing IntLiteralNode (non-pointer) in Value constraint")
		return ast.TypeNode{Ident: ast.TypeInt}, nil
	case *ast.FloatLiteralNode:
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
		}).Debugf("Processing FloatLiteralNode in Value constraint")
		return ast.TypeNode{Ident: ast.TypeFloat}, nil
	case ast.FloatLiteralNode:
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
		}).Debugf("Processing FloatLiteralNode (non-pointer) in Value constraint")
		return ast.TypeNode{Ident: ast.TypeFloat}, nil
	case *ast.BoolLiteralNode:
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
		}).Debugf("Processing BoolLiteralNode in Value constraint")
		return ast.TypeNode{Ident: ast.TypeBool}, nil
	case ast.BoolLiteralNode:
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
		}).Debugf("Processing BoolLiteralNode (non-pointer) in Value constraint")
		return ast.TypeNode{Ident: ast.TypeBool}, nil
	case *ast.NilLiteralNode:
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
		}).Debugf("Processing NilLiteralNode in Value constraint")
		// Handle Value(nil) constraints - use expectedType if provided, otherwise return a generic type
		if expectedType != nil {
			tc.log.WithFields(logrus.Fields{
				"function":     "inferValueConstraintType",
				"fieldName":    fieldName,
				"expectedType": expectedType.Ident,
			}).Debugf("Using expectedType for Value(nil) constraint")
			// If the expected type is already a pointer type, return it directly
			if expectedType.Ident == ast.TypePointer {
				return *expectedType, nil
			}
			// Otherwise, return a pointer to the expected type
			return ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{*expectedType}}, nil
		}
		// For nil values without expected type, return a pointer to a concrete type
		// Use String as a reasonable default for pointer types
		stringType := ast.TypeNode{Ident: ast.TypeString}
		return ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{stringType}}, nil
	case ast.NilLiteralNode:
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
		}).Debugf("Processing NilLiteralNode (non-pointer) in Value constraint")
		// Handle Value(nil) constraints - use expectedType if provided, otherwise return a generic type
		if expectedType != nil {
			tc.log.WithFields(logrus.Fields{
				"function":     "inferValueConstraintType",
				"fieldName":    fieldName,
				"expectedType": expectedType.Ident,
			}).Debugf("Using expectedType for Value(nil) constraint")
			// Return the expected type directly - it should already be the correct pointer type
			return *expectedType, nil
		}
		// For nil values without expected type, return a pointer to a concrete type
		// Use String as a reasonable default for pointer types
		stringType := ast.TypeNode{Ident: ast.TypeString}
		return ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{stringType}}, nil
	case *ast.ReferenceNode:
		// Handle Value(&variable) constraints
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
			"valueType": fmt.Sprintf("%T", v),
			"value":     fmt.Sprintf("%+v", v),
		}).Debugf("Processing ReferenceNode in Value constraint")

		if v.Value != nil {
			tc.log.WithFields(logrus.Fields{
				"function":   "inferValueConstraintType",
				"fieldName":  fieldName,
				"vValueType": fmt.Sprintf("%T", v.Value),
				"vValue":     fmt.Sprintf("%+v", v.Value),
			}).Debugf("ReferenceNode has non-nil Value")

			if varNode, ok := v.Value.(*ast.VariableNode); ok {
				tc.log.WithFields(logrus.Fields{
					"function":  "inferValueConstraintType",
					"fieldName": fieldName,
					"varNode":   fmt.Sprintf("%+v", varNode),
					"varIdent":  varNode.Ident.ID,
				}).Debugf("ReferenceNode Value is VariableNode")

				// Look up the variable's actual type
				tc.log.WithFields(logrus.Fields{
					"function":  "inferValueConstraintType",
					"fieldName": fieldName,
					"varIdent":  varNode.Ident.ID,
					"scope":     tc.CurrentScope().String(),
				}).Debugf("About to lookup variable type")

				varType, err := tc.LookupVariableType(varNode, tc.CurrentScope())
				if err == nil {
					tc.log.WithFields(logrus.Fields{
						"function":  "inferValueConstraintType",
						"fieldName": fieldName,
						"varType":   varType.Ident,
					}).Debugf("Successfully inferred type from referenced variable in Value constraint")
					return ast.TypeNode{
						Ident:      ast.TypePointer,
						TypeParams: []ast.TypeNode{varType},
					}, nil
				}
				tc.log.WithFields(logrus.Fields{
					"function":  "inferValueConstraintType",
					"fieldName": fieldName,
					"varIdent":  varNode.Ident.ID,
					"error":     err.Error(),
				}).Debugf("Failed to lookup variable type for ReferenceNode")
			} else {
				tc.log.WithFields(logrus.Fields{
					"function":   "inferValueConstraintType",
					"fieldName":  fieldName,
					"vValueType": fmt.Sprintf("%T", v.Value),
					"vValue":     fmt.Sprintf("%+v", v.Value),
				}).Debugf("ReferenceNode Value is not VariableNode")
			}
		} else {
			tc.log.WithFields(logrus.Fields{
				"function":  "inferValueConstraintType",
				"fieldName": fieldName,
			}).Debugf("ReferenceNode has nil Value")
		}

		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
			"valueType": fmt.Sprintf("%T", v),
		}).Debugf("Unsupported reference value type in Value constraint")
		// Fallback: use expectedType if provided
		if expectedType != nil {
			tc.log.WithFields(logrus.Fields{
				"function":     "inferValueConstraintType",
				"fieldName":    fieldName,
				"expectedType": expectedType.Ident,
			}).Debugf("Falling back to expectedType for Value constraint (non-pointer)")
			return *expectedType, nil
		}
		return ast.TypeNode{}, fmt.Errorf("could not infer type for Value constraint on field '%s'", fieldName)
	case ast.ReferenceNode:
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
			"valueType": fmt.Sprintf("%T", v),
			"value":     fmt.Sprintf("%+v", v),
		}).Debugf("Processing ReferenceNode (non-pointer) in Value constraint")
		// Handle Value(&variable) constraints (non-pointer version)
		if v.Value != nil {
			tc.log.WithFields(logrus.Fields{
				"function":   "inferValueConstraintType",
				"fieldName":  fieldName,
				"vValueType": fmt.Sprintf("%T", v.Value),
				"vValue":     fmt.Sprintf("%+v", v.Value),
			}).Debugf("ReferenceNode (non-pointer) has non-nil Value")

			// Try pointer version first
			if varNode, ok := v.Value.(*ast.VariableNode); ok {
				tc.log.WithFields(logrus.Fields{
					"function":  "inferValueConstraintType",
					"fieldName": fieldName,
					"varNode":   fmt.Sprintf("%+v", varNode),
					"varIdent":  varNode.Ident.ID,
				}).Debugf("ReferenceNode (non-pointer) Value is *VariableNode")

				// Look up the variable's actual type
				tc.log.WithFields(logrus.Fields{
					"function":  "inferValueConstraintType",
					"fieldName": fieldName,
					"varIdent":  varNode.Ident.ID,
					"scope":     tc.CurrentScope().String(),
				}).Debugf("About to lookup variable type (non-pointer)")

				varType, err := tc.LookupVariableType(varNode, tc.CurrentScope())
				if err == nil {
					tc.log.WithFields(logrus.Fields{
						"function":  "inferValueConstraintType",
						"fieldName": fieldName,
						"varType":   varType.Ident,
					}).Debugf("Successfully inferred type from referenced variable in Value constraint (non-pointer)")
					return ast.TypeNode{
						Ident:      ast.TypePointer,
						TypeParams: []ast.TypeNode{varType},
					}, nil
				}
				tc.log.WithFields(logrus.Fields{
					"function":  "inferValueConstraintType",
					"fieldName": fieldName,
					"varIdent":  varNode.Ident.ID,
					"error":     err.Error(),
				}).Debugf("Failed to lookup variable type for ReferenceNode (non-pointer)")
			} else if varNode, ok := v.Value.(ast.VariableNode); ok {
				// Try non-pointer version
				tc.log.WithFields(logrus.Fields{
					"function":  "inferValueConstraintType",
					"fieldName": fieldName,
					"varNode":   fmt.Sprintf("%+v", varNode),
					"varIdent":  varNode.Ident.ID,
				}).Debugf("ReferenceNode (non-pointer) Value is VariableNode (non-pointer)")

				// Look up the variable's actual type
				tc.log.WithFields(logrus.Fields{
					"function":  "inferValueConstraintType",
					"fieldName": fieldName,
					"varIdent":  varNode.Ident.ID,
					"scope":     tc.CurrentScope().String(),
				}).Debugf("About to lookup variable type (non-pointer, non-pointer var)")

				varType, err := tc.LookupVariableType(&varNode, tc.CurrentScope())
				if err != nil {
					tc.log.WithFields(logrus.Fields{
						"function":  "inferValueConstraintType",
						"fieldName": fieldName,
						"varIdent":  varNode.Ident.ID,
						"error":     err.Error(),
					}).Debugf("Failed to lookup variable type for ReferenceNode (non-pointer, non-pointer var)")
				} else {
					tc.log.WithFields(logrus.Fields{
						"function":  "inferValueConstraintType",
						"fieldName": fieldName,
						"varType":   varType.Ident,
					}).Debugf("Successfully inferred type from referenced variable in Value constraint (non-pointer, non-pointer var)")
					return ast.TypeNode{
						Ident:      ast.TypePointer,
						TypeParams: []ast.TypeNode{varType},
					}, nil
				}
			} else {
				tc.log.WithFields(logrus.Fields{
					"function":   "inferValueConstraintType",
					"fieldName":  fieldName,
					"vValueType": fmt.Sprintf("%T", v.Value),
					"vValue":     fmt.Sprintf("%+v", v.Value),
				}).Debugf("ReferenceNode (non-pointer) Value is not VariableNode")
			}
		} else {
			tc.log.WithFields(logrus.Fields{
				"function":  "inferValueConstraintType",
				"fieldName": fieldName,
			}).Debugf("ReferenceNode (non-pointer) has nil Value")
		}

		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
			"valueType": fmt.Sprintf("%T", v),
		}).Debugf("Unsupported reference value type in Value constraint (non-pointer)")
		// Fallback: use expectedType if provided
		if expectedType != nil {
			tc.log.WithFields(logrus.Fields{
				"function":     "inferValueConstraintType",
				"fieldName":    fieldName,
				"expectedType": expectedType.Ident,
			}).Debugf("Falling back to expectedType for Value constraint (non-pointer)")
			return *expectedType, nil
		}
		return ast.TypeNode{}, fmt.Errorf("could not infer type for Value constraint on field '%s'", fieldName)
	default:
		tc.log.WithFields(logrus.Fields{
			"function":  "inferValueConstraintType",
			"fieldName": fieldName,
			"valueType": fmt.Sprintf("%T", v),
			"value":     fmt.Sprintf("%+v", v),
		}).Debugf("Unsupported value type in Value constraint")
		return ast.TypeNode{}, fmt.Errorf("could not infer type for Value constraint on field '%s'", fieldName)
	}
}
