package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"
	"reflect"
	"strconv"
	"strings"
)

// negateCondition negates a condition
func negateCondition(condition goast.Expr) goast.Expr {
	return &goast.UnaryExpr{
		Op: token.NOT,
		X:  condition,
	}
}

// disjoin joins a list of conditions with OR ("any condition must match")
func disjoin(conditions []goast.Expr) goast.Expr {
	if len(conditions) == 0 {
		return &goast.Ident{Name: BoolConstantFalse}
	}
	combined := conditions[0]
	for i := 1; i < len(conditions); i++ {
		combined = &goast.BinaryExpr{
			X:  combined,
			Op: token.LOR,
			Y:  conditions[i],
		}
	}
	return combined
}

// conjoin joins a list of conditions with AND ("all conditions must match")
func conjoin(conditions []goast.Expr) goast.Expr {
	if len(conditions) == 0 {
		return &goast.Ident{Name: BoolConstantFalse}
	}
	combined := conditions[0]
	for i := 1; i < len(conditions); i++ {
		combined = &goast.BinaryExpr{
			X:  combined,
			Op: token.LAND,
			Y:  conditions[i],
		}
	}
	return combined
}

func (t *Transformer) transformOperator(op ast.TokenIdent) (token.Token, error) {
	switch op {
	case ast.TokenPlus:
		return token.ADD, nil
	case ast.TokenMinus:
		return token.SUB, nil
	case ast.TokenStar:
		return token.MUL, nil
	case ast.TokenDivide:
		return token.QUO, nil
	case ast.TokenModulo:
		return token.REM, nil
	case ast.TokenEquals:
		return token.EQL, nil
	case ast.TokenNotEquals:
		return token.NEQ, nil
	case ast.TokenGreater:
		return token.GTR, nil
	case ast.TokenLess:
		return token.LSS, nil
	case ast.TokenGreaterEqual:
		return token.GEQ, nil
	case ast.TokenLessEqual:
		return token.LEQ, nil
	case ast.TokenLogicalAnd:
		return token.LAND, nil
	case ast.TokenLogicalOr:
		return token.LOR, nil
	case ast.TokenLogicalNot:
		return token.NOT, nil
	}

	return 0, fmt.Errorf("unsupported operator: %s", op)
}

func (t *Transformer) transformExpression(expr ast.ExpressionNode) (goast.Expr, error) {
	switch e := expr.(type) {
	case ast.IntLiteralNode:
		return &goast.BasicLit{
			Kind:  token.INT,
			Value: strconv.FormatInt(e.Value, 10),
		}, nil
	case ast.FloatLiteralNode:
		return &goast.BasicLit{
			Kind:  token.FLOAT,
			Value: strconv.FormatFloat(e.Value, 'f', -1, 64),
		}, nil
	case ast.StringLiteralNode:
		return &goast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(e.Value),
		}, nil
	case ast.BoolLiteralNode:
		if e.Value {
			return goast.NewIdent("true"), nil
		}
		return goast.NewIdent("false"), nil
	case ast.NilLiteralNode:
		return goast.NewIdent("nil"), nil
	case ast.UnaryExpressionNode:
		op, err := t.transformOperator(e.Operator)
		if err != nil {
			return nil, err
		}
		expr, err := t.transformExpression(e.Operand)
		if err != nil {
			return nil, err
		}
		return &goast.UnaryExpr{
			Op: op,
			X:  expr,
		}, nil
	case ast.BinaryExpressionNode:
		left, err := t.transformExpression(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := t.transformExpression(e.Right)
		if err != nil {
			return nil, err
		}
		op, err := t.transformOperator(e.Operator)
		if err != nil {
			return nil, err
		}
		return &goast.BinaryExpr{
			X:  left,
			Op: op,
			Y:  right,
		}, nil
	case ast.VariableNode:
		// Support field access: split by '.' and capitalize each field after the first
		parts := strings.Split(e.GetIdent(), ".")
		if len(parts) == 1 {
			return &goast.Ident{Name: parts[0]}, nil
		}
		var sel goast.Expr = goast.NewIdent(parts[0])
		for _, field := range parts[1:] {
			fieldName := field
			if t.ExportReturnStructFields {
				fieldName = capitalizeFirst(field)
			}
			sel = &goast.SelectorExpr{
				X:   sel,
				Sel: goast.NewIdent(fieldName),
			}
		}
		return sel, nil
	case ast.FunctionCallNode:
		// Look up parameter types for the function
		paramTypes := make([]ast.TypeNode, len(e.Arguments))
		if sig, ok := t.TypeChecker.Functions[e.Function.ID]; ok && len(sig.Parameters) == len(e.Arguments) {
			for i, param := range sig.Parameters {
				if param.Type.Ident == ast.TypeAssertion && param.Type.Assertion != nil {
					inferredTypes, err := t.TypeChecker.InferAssertionType(param.Type.Assertion, false, "", nil)
					if err == nil && len(inferredTypes) > 0 {
						paramTypes[i] = inferredTypes[0]
					} else {
						paramTypes[i] = param.Type
					}
				} else {
					paramTypes[i] = param.Type
				}
			}
		}
		args := make([]goast.Expr, len(e.Arguments))
		for i, arg := range e.Arguments {
			if shapeArg, ok := arg.(ast.ShapeNode); ok && paramTypes[i].Ident != ast.TypeImplicit {
				// Use the unified helper to determine the expected type
				context := &ShapeContext{
					ExpectedType:   &paramTypes[i],
					FunctionName:   string(e.Function.ID),
					ParameterIndex: i,
				}
				expectedTypeForShape := t.getExpectedTypeForShape(&shapeArg, context)
				argExpr, err := t.transformShapeNodeWithExpectedType(&shapeArg, expectedTypeForShape)
				if err != nil {
					return nil, err
				}
				args[i] = argExpr
			} else {
				argExpr, err := t.transformExpression(arg)
				if err != nil {
					return nil, err
				}
				args[i] = argExpr
			}
		}
		return &goast.CallExpr{
			Fun:  goast.NewIdent(string(e.Function.ID)),
			Args: args,
		}, nil
	case ast.ReferenceNode:
		expr, err := t.transformExpression(e.Value)
		if err != nil {
			return nil, err
		}
		// Check if the inner expression is a struct literal (CompositeLit)
		if _, isStructLiteral := expr.(*goast.CompositeLit); isStructLiteral {
			// For struct literals, just return the struct literal
			// The outer context will handle adding the & if needed
			return expr, nil
		}
		// For other expressions, add the & operator
		return &goast.UnaryExpr{
			Op: token.AND,
			X:  expr,
		}, nil
	case ast.DereferenceNode:
		expr, err := t.transformExpression(e.Value)
		if err != nil {
			return nil, err
		}
		return &goast.UnaryExpr{
			Op: token.MUL,
			X:  expr,
		}, nil
	case ast.ShapeNode:
		// Use the unified helper to determine the expected type
		context := &ShapeContext{}
		expectedType := t.getExpectedTypeForShape(&e, context)
		// Always use the unified aliasing logic for shape literals
		return t.transformShapeNodeWithExpectedType(&e, expectedType)
	}

	return nil, fmt.Errorf("unsupported expression type: %s", reflect.TypeOf(expr).String())
}

// transformAssertionValue transforms an assertion value to a Go expression
func (t *Transformer) transformAssertionValue(assertion *ast.AssertionNode, expectedType *ast.TypeNode) (goast.Expr, error) {
	t.log.WithFields(map[string]interface{}{
		"assertionType":  fmt.Sprintf("%T", assertion),
		"assertionValue": fmt.Sprintf("%#v", assertion),
		"expectedType":   expectedType,
		"function":       "transformAssertionValue",
	}).Debug("Processing assertion value")

	// Check if this is a Value assertion with a value argument
	if len(assertion.Constraints) > 0 {
		constraint := assertion.Constraints[0]
		if constraint.Name == ast.ValueConstraint && len(constraint.Args) > 0 {
			arg := constraint.Args[0]
			if arg.Value != nil {
				// For Value assertions, we need to handle the value appropriately
				switch v := (*arg.Value).(type) {
				case ast.ReferenceNode:
					t.log.WithFields(map[string]interface{}{
						"expectedType": expectedType,
						"function":     "transformAssertionValue",
						"valueType":    fmt.Sprintf("%T", v.Value),
						"value":        fmt.Sprintf("%#v", v.Value),
					}).Debug("Value assertion with ReferenceNode")

					// For Value(Ref(x)), we want a pointer to x when used in a pointer context
					// So we transform the inner value and take its address
					innerExpr, err := t.transformExpression(v.Value)
					if err != nil {
						return nil, err
					}

					t.log.WithFields(map[string]interface{}{
						"expectedType":   expectedType,
						"function":       "transformAssertionValue",
						"innerExprType":  fmt.Sprintf("%T", innerExpr),
						"innerExprValue": fmt.Sprintf("%#v", innerExpr),
					}).Debug("After transforming inner expression of ReferenceNode")

					isStructLiteral := false
					if _, ok := innerExpr.(*goast.CompositeLit); ok {
						isStructLiteral = true
					}
					isIdentifier := false
					if _, ok := innerExpr.(*goast.Ident); ok {
						isIdentifier = true
					}
					if expectedType != nil && expectedType.Ident == ast.TypePointer {
						t.log.WithFields(map[string]interface{}{
							"expectedType":    expectedType.Ident,
							"function":        "transformAssertionValue",
							"innerExprType":   fmt.Sprintf("%T", innerExpr),
							"isStructLiteral": isStructLiteral,
							"isIdentifier":    isIdentifier,
						}).Debug("ReferenceNode in pointer context")
						// Only add & if not already a pointer, not a struct literal, and not an identifier
						if !isStructLiteral && !isIdentifier {
							t.log.WithFields(map[string]interface{}{
								"function": "transformAssertionValue",
							}).Debug("Adding & to inner expression")
							return &goast.UnaryExpr{Op: token.AND, X: innerExpr}, nil
						}
						// If struct literal or identifier, just return it (outer context will add & if needed)
						t.log.WithFields(map[string]interface{}{
							"function": "transformAssertionValue",
						}).Debug("Returning inner expression as-is (struct literal or identifier)")
						return innerExpr, nil
					}
					t.log.WithFields(map[string]interface{}{
						"function":      "transformAssertionValue",
						"innerExprType": fmt.Sprintf("%T", innerExpr),
					}).Debug("ReferenceNode not in pointer context, returning innerExpr as-is")
					return innerExpr, nil
				default:
					// For other value types, transform normally
					t.log.WithFields(map[string]interface{}{
						"valueType": fmt.Sprintf("%T", *arg.Value),
						"function":  "transformAssertionValue",
					}).Debug("Value assertion with non-ReferenceNode value")
					return t.transformExpression(*arg.Value)
				}
			}
		}
	}

	// For other assertion types, return a zero value based on the expected type
	// This is a fallback - in practice, we should handle more assertion types
	return goast.NewIdent("nil"), nil
}

// transformShapeNodeWithExpectedType generates a struct literal using the expected type if possible
func (t *Transformer) transformShapeNodeWithExpectedType(shape *ast.ShapeNode, expectedType *ast.TypeNode) (goast.Expr, error) {
	t.log.WithFields(map[string]interface{}{
		"expectedType": expectedType,
		"shape":        fmt.Sprintf("%+v", shape),
		"function":     "transformShapeNodeWithExpectedType",
		"baseType":     shape.BaseType,
		"fields":       fmt.Sprintf("%+v", shape.Fields),
	}).Debug("[DEBUG] Starting transformShapeNodeWithExpectedType")

	// Get the struct type to use for the composite literal
	var structType goast.Expr
	var fieldTypes map[string]string

	if expectedType != nil {
		t.log.WithFields(map[string]interface{}{
			"expectedType":      expectedType,
			"expectedTypeIdent": expectedType.Ident,
			"function":          "transformShapeNodeWithExpectedType",
		}).Debug("[DEBUG] Processing with expectedType")

		// Try to find an existing type that matches this shape
		typeIdent, found := t.findExistingTypeForShape(shape, expectedType)
		if found {
			t.log.WithFields(map[string]interface{}{
				"typeIdent": typeIdent,
				"function":  "transformShapeNodeWithExpectedType",
			}).Debug("[DEBUG] Found existing type for shape with expectedType")
			structType = goast.NewIdent(string(typeIdent))
		} else {
			t.log.WithFields(map[string]interface{}{
				"function": "transformShapeNodeWithExpectedType",
			}).Debug("[DEBUG] No existing type found for shape with expectedType, will use expectedType directly")
			structType = goast.NewIdent(string(expectedType.Ident))
		}

		// Extract field types from the expected type
		if def, ok := t.TypeChecker.Defs[expectedType.Ident]; ok {
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
					fieldTypes = make(map[string]string)
					for fieldName, field := range shapeExpr.Shape.Fields {
						if field.Type != nil {
							aliasName, err := t.TypeChecker.GetAliasedTypeName(*field.Type)
							if err == nil && aliasName != "" {
								fieldTypes[fieldName] = aliasName
							} else {
								fieldTypes[fieldName] = string(field.Type.Ident)
							}
						}
					}
					t.log.WithFields(map[string]interface{}{
						"fieldTypes": fieldTypes,
						"function":   "transformShapeNodeWithExpectedType",
					}).Debug("[DEBUG] Extracted field types from expected type")
				}
			}
		}
	} else {
		t.log.WithFields(map[string]interface{}{
			"function": "transformShapeNodeWithExpectedType",
		}).Debug("[DEBUG] No expectedType provided")

		// Try to find an existing type that matches this shape
		typeIdent, found := t.findExistingTypeForShape(shape, nil)
		if found {
			t.log.WithFields(map[string]interface{}{
				"typeIdent": typeIdent,
				"function":  "transformShapeNodeWithExpectedType",
			}).Debug("[DEBUG] Found existing type for shape without expectedType")
			structType = goast.NewIdent(string(typeIdent))
		} else {
			t.log.WithFields(map[string]interface{}{
				"function": "transformShapeNodeWithExpectedType",
			}).Debug("[DEBUG] No existing type found for shape without expectedType, will use hash-based type")
			// Generate a hash-based type name
			hash, err := t.TypeChecker.Hasher.HashNode(shape)
			if err != nil {
				return nil, fmt.Errorf("failed to hash shape: %v", err)
			}
			typeName := hash.ToTypeIdent()
			structType = goast.NewIdent(string(typeName))
			t.log.WithFields(map[string]interface{}{
				"hash":     hash,
				"typeName": typeName,
				"function": "transformShapeNodeWithExpectedType",
			}).Debug("[DEBUG] Generated hash-based type name")
		}
	}

	t.log.WithFields(map[string]interface{}{
		"structType": fmt.Sprintf("%#v", structType),
		"function":   "transformShapeNodeWithExpectedType",
	}).Debug("[DEBUG] Final structType chosen for Go code generation")

	// Create the composite literal
	fields := make([]*goast.KeyValueExpr, 0, len(shape.Fields))
	for fieldName, field := range shape.Fields {
		t.log.WithFields(map[string]interface{}{
			"fieldName": fieldName,
			"field":     fmt.Sprintf("%+v", field),
			"function":  "transformShapeNodeWithExpectedType",
		}).Debug("[DEBUG] Processing field in struct literal")

		var fieldValue goast.Expr
		var err error

		if field.Node != nil {
			t.log.WithFields(map[string]interface{}{
				"fieldName": fieldName,
				"function":  "transformShapeNodeWithExpectedType",
			}).Debug("[DEBUG] Field has Node, transforming it")
			fieldValue, err = t.transformExpression(field.Node.(ast.ExpressionNode))
		} else if field.Shape != nil {
			t.log.WithFields(map[string]interface{}{
				"fieldName": fieldName,
				"function":  "transformShapeNodeWithExpectedType",
			}).Debug("[DEBUG] Field has Shape, transforming it")
			fieldValue, err = t.transformShapeNodeWithExpectedType(field.Shape, nil)
		} else if field.Assertion != nil {
			t.log.WithFields(map[string]interface{}{
				"fieldName": fieldName,
				"function":  "transformShapeNodeWithExpectedType",
			}).Debug("[DEBUG] Field has Assertion, transforming it")
			fieldValue, err = t.transformAssertionValue(field.Assertion, nil)
		} else if field.Type != nil {
			t.log.WithFields(map[string]interface{}{
				"fieldName": fieldName,
				"fieldType": field.Type,
				"function":  "transformShapeNodeWithExpectedType",
			}).Debug("[DEBUG] Field has Type only, checking for struct type")

			// Handle pointer types correctly
			if field.Type.Ident == ast.TypePointer && len(field.Type.TypeParams) > 0 {
				// For pointer types, emit nil
				fieldValue = goast.NewIdent("nil")
			} else {
				// If the field is a struct type, emit zero value (TypeName{})
				aliasName, err := t.TypeChecker.GetAliasedTypeName(*field.Type)
				if err == nil && aliasName != "" {
					t.log.WithFields(map[string]interface{}{
						"fieldName": fieldName,
						"aliasName": aliasName,
						"function":  "transformShapeNodeWithExpectedType",
					}).Debug("[DEBUG] Emitting zero value for struct type field")

					// Emit Go zero value for primitives
					switch aliasName {
					case "int":
						fieldValue = &goast.BasicLit{Kind: token.INT, Value: "0"}
					case "string":
						fieldValue = &goast.BasicLit{Kind: token.STRING, Value: "\"\""}
					case "bool":
						fieldValue = goast.NewIdent("false")
					case "float64":
						fieldValue = &goast.BasicLit{Kind: token.FLOAT, Value: "0.0"}
					default:
						// Check if it's a hash-based type (starts with T_)
						if strings.HasPrefix(aliasName, "T_") {
							// For hash-based types, use nil instead
							fieldValue = goast.NewIdent("nil")
						} else {
							fieldValue = &goast.CompositeLit{
								Type: goast.NewIdent(aliasName),
							}
						}
					}
				} else {
					t.log.WithFields(map[string]interface{}{
						"fieldName": fieldName,
						"function":  "transformShapeNodeWithExpectedType",
					}).Debug("[DEBUG] No alias found for field type, using nil")
					fieldValue = goast.NewIdent("nil")
				}
			}
		} else {
			t.log.WithFields(map[string]interface{}{
				"fieldName": fieldName,
				"function":  "transformShapeNodeWithExpectedType",
			}).Debug("[DEBUG] Field has no Node/Shape/Assertion/Type, using nil")
			fieldValue = goast.NewIdent("nil")
		}

		if err != nil {
			return nil, fmt.Errorf("failed to transform field %s: %v", fieldName, err)
		}

		// Capitalize field name for Go struct literal if needed
		goFieldName := fieldName
		if t.ExportReturnStructFields {
			goFieldName = capitalizeFirst(fieldName)
		}
		t.log.WithFields(map[string]interface{}{
			"forstField": fieldName,
			"goField":    goFieldName,
			"function":   "transformShapeNodeWithExpectedType",
		}).Debug("[DEBUG] Mapping Forst field to Go field in struct literal")

		fields = append(fields, &goast.KeyValueExpr{
			Key:   goast.NewIdent(goFieldName),
			Value: fieldValue,
		})
	}

	return &goast.CompositeLit{
		Type: structType,
		Elts: func() []goast.Expr {
			exprs := make([]goast.Expr, len(fields))
			for i, field := range fields {
				exprs[i] = field
			}
			return exprs
		}(),
	}, nil
}

// exprToTypeName extracts the type name from a go/ast.Expr
func (t *Transformer) exprToTypeName(expr goast.Expr) string {
	switch e := expr.(type) {
	case *goast.Ident:
		return e.Name
	case *goast.StarExpr:
		return "*" + t.exprToTypeName(e.X)
	case *goast.SelectorExpr:
		return e.Sel.Name
	case *goast.StructType:
		return "struct" // anonymous
	default:
		return "" // unknown
	}
}
