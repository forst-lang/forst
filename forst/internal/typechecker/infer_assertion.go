package typechecker

import (
	"fmt"
	"forst/internal/ast"
	"time"

	logrus "github.com/sirupsen/logrus"
)

// TODO: Improve assertion type inference
// This should handle:
// 1. Complex constraints
// 2. Nested assertions
// 3. Type aliases
// 4. Generic types
func (tc *TypeChecker) inferAssertionType(assertion *ast.AssertionNode, isTypeGuard bool) ([]ast.TypeNode, error) {
	if assertion == nil {
		return nil, nil
	}

	tc.log.WithFields(logrus.Fields{
		"function":  "inferAssertionType",
		"assertion": assertion,
	}).Tracef("Inferring type for assertion")

	// First, get the base type if it exists
	var baseType ast.TypeIdent
	if assertion.BaseType != nil {
		baseType = *assertion.BaseType
		tc.log.WithFields(logrus.Fields{
			"function": "inferAssertionType",
			"baseType": baseType,
		}).Tracef("Assertion has base type")
	}

	// Create a new shape type to hold the merged fields
	mergedFields := make(map[string]ast.ShapeFieldNode)

	// If we have a base type, get its fields first
	if baseType != "" {
		if def, exists := tc.Defs[baseType]; exists {
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
					for k, v := range shapeExpr.Shape.Fields {
						mergedFields[k] = v
						tc.log.WithFields(logrus.Fields{
							"function": "inferAssertionType",
							"field":    k,
							"value":    v,
						}).Tracef("Added field from base type")
					}
				} else if assertionExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
					if assertionExpr.Assertion != nil {
						baseFields := tc.resolveShapeFieldsFromAssertion(assertionExpr.Assertion)
						for k, v := range baseFields {
							mergedFields[k] = v
							tc.log.WithFields(logrus.Fields{
								"function": "inferAssertionType",
								"field":    k,
								"value":    v,
							}).Tracef("Added field from base assertion")
						}
					}
				}
			}
		}
	}

	// Process each constraint
	for _, constraint := range assertion.Constraints {
		tc.log.WithFields(logrus.Fields{
			"function":   "inferAssertionType",
			"constraint": constraint.Name,
		}).Tracef("Processing constraint")

		// Get the type guard definition
		guardDef, exists := tc.Defs[ast.TypeIdent(constraint.Name)]
		if !exists {
			return nil, fmt.Errorf("type guard %s not found", constraint.Name)
		}

		// Push a new scope for the type guard's body
		var guardNode ast.TypeGuardNode
		if ptr, ok := guardDef.(*ast.TypeGuardNode); ok {
			guardNode = *ptr
		} else {
			guardNode = guardDef.(ast.TypeGuardNode)
		}

		tc.log.WithFields(logrus.Fields{
			"function":    "inferAssertionType",
			"subject":     guardNode.Subject.GetIdent(),
			"subjectType": guardNode.Subject.GetType().Ident,
			"parameters":  guardNode.Parameters(),
		}).Tracef("Subject parameter and additional parameters")

		// Map constraint arguments to parameters
		argMap := make(map[string]ast.Node)
		for i, arg := range constraint.Args {
			if i+1 < len(guardNode.Parameters()) {
				param := guardNode.Parameters()[i+1] // +1 to skip subject parameter
				argMap[param.GetIdent()] = arg
				tc.log.WithFields(logrus.Fields{
					"function": "inferAssertionType",
					"param":    param.GetIdent(),
					"arg":      fmt.Sprintf("%+v", arg),
				}).Tracef("Mapping parameter to argument")
			}
		}

		// TODO: Fix this hardcoded Input constraint
		if constraint.Name == "Input" {
			// For Input constraint, we want to keep existing fields and add new input fields
			// First, preserve the ctx field if it exists
			// TODO: Fix this hardcoded ctx field
			if ctxField, hasCtx := mergedFields["ctx"]; hasCtx {
				// Create a temporary map to store fields
				tempFields := make(map[string]ast.ShapeFieldNode)
				// TODO: Fix this hardcoded ctx field
				tempFields["ctx"] = ctxField

				// Add the input field from the argument
				// TODO: Fix this hardcoded input field
				if arg, ok := argMap["input"]; ok {
					if argNode, ok := arg.(ast.ConstraintArgumentNode); ok {
						if argNode.Shape != nil {
							// Ensure the shape type is inferred and registered
							if _, err := tc.inferShapeType(*argNode.Shape); err != nil {
								return nil, err
							}
							for k, v := range argNode.Shape.Fields {
								tempFields[k] = v
								tc.log.WithFields(logrus.Fields{
									"function": "inferAssertionType",
									"field":    k,
									"value":    v,
								}).Tracef("Added field from mutation input")
							}
						}
					}
				}

				// Replace mergedFields with our new map that preserves ctx
				mergedFields = tempFields
			} else {
				// If no ctx field exists, just add the input fields
				// TODO: Fix this hardcoded input field
				if arg, ok := argMap["input"]; ok {
					if argNode, ok := arg.(ast.ConstraintArgumentNode); ok {
						if argNode.Shape != nil {
							// Ensure the shape type is inferred and registered
							if _, err := tc.inferShapeType(*argNode.Shape); err != nil {
								return nil, err
							}
							for k, v := range argNode.Shape.Fields {
								mergedFields[k] = v
								tc.log.WithFields(logrus.Fields{
									"function": "inferAssertionType",
									"field":    k,
									"value":    v,
								}).Tracef("Added field from mutation input")
							}
						}
					}
				}
			}
			continue
		}

		// For other constraints, process each parameter
		for _, param := range guardNode.Parameters() {
			if param.GetIdent() != guardNode.Subject.GetIdent() {
				// Add the field from the parameter
				paramType := param.GetType().Ident
				mergedFields[param.GetIdent()] = ast.ShapeFieldNode{
					Assertion: &ast.AssertionNode{
						BaseType: &paramType,
					},
				}

				// If we have an argument for this parameter, use its concrete type
				if arg, ok := argMap[param.GetIdent()]; ok {
					if argNode, ok := arg.(ast.ConstraintArgumentNode); ok {
						if argNode.Shape != nil {
							// Ensure the shape type is inferred and registered
							if _, err := tc.inferShapeType(*argNode.Shape); err != nil {
								return nil, err
							}
							// If it's a shape, merge its fields
							for k, v := range argNode.Shape.Fields {
								mergedFields[k] = v
								tc.log.WithFields(logrus.Fields{
									"function": "inferAssertionType",
									"field":    k,
									"value":    v,
								}).Tracef("Added field from shape argument")
							}
						} else if argNode.Type != nil {
							// Use the concrete type from the argument instead of the generic Shape
							argType := argNode.Type.Ident
							mergedFields[param.GetIdent()] = ast.ShapeFieldNode{
								Assertion: &ast.AssertionNode{
									BaseType: &argType,
								},
							}
							tc.log.WithFields(logrus.Fields{
								"function": "inferAssertionType",
								"field":    param.GetIdent(),
								"value":    argNode.Type.Ident,
							}).Tracef("Added field with concrete type")
						}
					}
				}
			}
		}
	}

	// Use the structural hash for this assertion node
	hash, err := tc.Hasher.HashNode(assertion)
	if err != nil {
		return nil, fmt.Errorf("failed to hash assertion during inferAssertionType: %s", err)
	}
	typeIdent := hash.ToTypeIdent()
	tc.log.WithFields(logrus.Fields{
		"function":  "inferAssertionType",
		"assertion": assertion,
		"fields":    mergedFields,
		"typeIdent": typeIdent,
	}).Tracef("Stored shape type with fields")

	// Add detailed debugging for field storage
	tc.log.WithFields(logrus.Fields{
		"function":  "inferAssertionType",
		"typeIdent": typeIdent,
	}).Debugf("=== DETAILED FIELD DEBUG ===")
	for fieldName, field := range mergedFields {
		tc.log.WithFields(logrus.Fields{
			"function":  "inferAssertionType",
			"typeIdent": typeIdent,
			"fieldName": fieldName,
			"field":     fmt.Sprintf("%+v", field),
		}).Debugf("Field: %s", fieldName)
	}
	tc.log.WithFields(logrus.Fields{
		"function":  "inferAssertionType",
		"typeIdent": typeIdent,
	}).Debugf("=== END FIELD DEBUG ===")

	// Store the shape type
	tc.Defs[typeIdent] = ast.TypeDefNode{
		Ident: typeIdent,
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: mergedFields,
			},
		},
	}

	shapeType := ast.TypeNode{Ident: typeIdent}
	tc.storeInferredType(assertion, []ast.TypeNode{shapeType})
	return []ast.TypeNode{shapeType}, nil
}

func generateUniqueID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
