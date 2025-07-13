package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"
	"reflect"
	"sort"
	"strconv"
	"strings"

	logrus "github.com/sirupsen/logrus"
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
				argExpr, err := t.transformShapeNodeWithExpectedType(&shapeArg, &paramTypes[i])
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
		// Try to find a matching named type for this shape
		typeIdent, found := t.findExistingTypeForShape(&e, nil)
		var expectedType *ast.TypeNode
		if found && typeIdent != "" {
			expectedType = &ast.TypeNode{Ident: typeIdent}
			t.log.WithFields(map[string]interface{}{
				"typeIdent": typeIdent,
				"function":  "transformExpression-ShapeNode",
			}).Debug("[PATCH] Found matching named type for shape literal")
		}
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
	}).Debug("Starting transformShapeNodeWithExpectedType")

	// Get the struct type to use for the composite literal
	var structType goast.Expr
	var fieldTypes map[string]string

	// Always use the unified aliasing logic for the expected type if present
	if expectedType != nil {
		if expectedType.Ident == ast.TypePointer && len(expectedType.TypeParams) > 0 {
			// Use the base type name for pointer types
			aliasName, err := t.TypeChecker.GetAliasedTypeName(expectedType.TypeParams[0])
			if err == nil && aliasName != "" {
				structType = goast.NewIdent(aliasName)
			}
		} else {
			aliasName, err := t.TypeChecker.GetAliasedTypeName(*expectedType)
			if err == nil && aliasName != "" {
				structType = goast.NewIdent(aliasName)
			}
		}
	}

	// Always prefer typeIdent if a shape match is found
	t.log.WithFields(logrus.Fields{
		"function":     "transformShapeNodeWithExpectedType",
		"shape":        fmt.Sprintf("%+v", shape),
		"expectedType": expectedType,
	}).Debug("Calling findExistingTypeForShape")
	typeIdent, found := t.findExistingTypeForShape(shape, expectedType)
	t.log.WithFields(logrus.Fields{
		"function":  "transformShapeNodeWithExpectedType",
		"typeIdent": typeIdent,
		"found":     found,
	}).Debug("findExistingTypeForShape result")

	// PATCH: If a named type is found, always use it for structType
	if found && typeIdent != "" {
		structType = goast.NewIdent(string(typeIdent))
		t.log.WithFields(map[string]interface{}{
			"typeIdent":  typeIdent,
			"structType": structType,
			"function":   "transformShapeNodeWithExpectedType",
		}).Debug("[PATCH] Using named structType for Go code generation")
	}

	// If we still don't have a struct type, use the hash-based type but check for an alias
	if structType == nil {
		hash, err := t.TypeChecker.Hasher.HashNode(*shape)
		if err != nil {
			return nil, fmt.Errorf("failed to hash shape: %w", err)
		}
		hashTypeName := string(hash.ToTypeIdent())
		aliasName, err := t.TypeChecker.GetAliasedTypeName(ast.TypeNode{Ident: ast.TypeIdent(hashTypeName)})
		t.log.WithFields(map[string]interface{}{
			"function":     "transformShapeNodeWithExpectedType",
			"hashTypeName": hashTypeName,
			"aliasName":    aliasName,
			"err":          err,
		}).Debug("Resolving hash-based type to aliased type")
		if err == nil && aliasName != "" {
			structType = goast.NewIdent(aliasName)
		} else {
			structType = goast.NewIdent(hashTypeName)
		}
	}

	// Add debug log after determining structType
	if structType != nil {
		t.log.WithFields(map[string]interface{}{
			"function":   "transformShapeNodeWithExpectedType",
			"structType": fmt.Sprintf("%#v", structType),
		}).Debug("[DEBUG] Using structType for Go code generation")
	}

	// Extract field types from the expected type definition
	if expectedType != nil {
		if def, exists := t.TypeChecker.Defs[expectedType.Ident]; exists {
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
					fieldTypes = make(map[string]string)
					for fieldName, field := range shapeExpr.Shape.Fields {
						if field.Type != nil {
							// Handle pointer types by extracting the base type from TypeParams
							if field.Type.Ident == ast.TypePointer && len(field.Type.TypeParams) > 0 {
								fieldTypes[fieldName] = "*" + string(field.Type.TypeParams[0].Ident)
							} else {
								fieldTypes[fieldName] = string(field.Type.Ident)
							}
						} else if field.Shape != nil {
							// For nested shapes, try to resolve the expected type
							// Look up the shape in the typechecker to find its type
							if shapeHash, err := t.TypeChecker.Hasher.HashNode(*field.Shape); err == nil {
								shapeTypeIdent := shapeHash.ToTypeIdent()
								if _, exists := t.TypeChecker.Defs[shapeTypeIdent]; exists {
									fieldTypes[fieldName] = string(shapeTypeIdent)
								} else {
									// If not found, use "Shape" as fallback
									fieldTypes[fieldName] = "Shape"
								}
							} else {
								fieldTypes[fieldName] = "Shape"
							}
						} else if field.Assertion != nil {
							// Handle assertion types by extracting the base type
							if field.Assertion.BaseType != nil {
								fieldTypes[fieldName] = string(*field.Assertion.BaseType)
							} else {
								// If no base type, use "Shape" as fallback
								fieldTypes[fieldName] = "Shape"
							}
						}
					}
				}
			}
		}

		// Build the struct literal fields, recursively using expected field types
		fields := []goast.Expr{}

		// Canonicalize field order for deterministic emission
		fieldNames := make([]string, 0, len(shape.Fields))
		for name := range shape.Fields {
			fieldNames = append(fieldNames, name)
		}
		sort.Strings(fieldNames)

		for _, name := range fieldNames {
			field := shape.Fields[name]
			var fieldValue goast.Expr
			var err error
			// If we have an expected field type, use it recursively
			var expectedFieldType *ast.TypeNode
			if fieldTypes != nil {
				fieldTypeName := fieldTypes[name]
				if strings.HasPrefix(fieldTypeName, "*") {
					expectedFieldType = &ast.TypeNode{
						Ident:      ast.TypePointer,
						TypeParams: []ast.TypeNode{{Ident: ast.TypeIdent(fieldTypeName[1:])}},
					}
				} else {
					expectedFieldType = &ast.TypeNode{Ident: ast.TypeIdent(fieldTypeName)}
				}
			}

			// Handle field.Node if present
			if field.Node != nil {
				switch n := field.Node.(type) {
				case *ast.ShapeNode:
					// If expectedFieldType is a pointer, use its base type for the inner struct
					innerExpected := expectedFieldType
					if expectedFieldType != nil && expectedFieldType.Ident == ast.TypePointer && len(expectedFieldType.TypeParams) > 0 {
						innerExpected = &expectedFieldType.TypeParams[0]
					}
					fieldValue, err = t.transformShapeNodeWithExpectedType(n, innerExpected)
				case ast.ShapeNode:
					innerExpected := expectedFieldType
					if expectedFieldType != nil && expectedFieldType.Ident == ast.TypePointer && len(expectedFieldType.TypeParams) > 0 {
						innerExpected = &expectedFieldType.TypeParams[0]
					}
					fieldValue, err = t.transformShapeNodeWithExpectedType(&n, innerExpected)
				case ast.ExpressionNode:
					fieldValue, err = t.transformExpression(n)
				default:
					fieldValue = goast.NewIdent("nil") // fallback for unsupported node types
				}
			} else if field.Shape != nil {
				// If expectedFieldType is a pointer, use its base type for the inner struct
				innerExpected := expectedFieldType
				if expectedFieldType != nil && expectedFieldType.Ident == ast.TypePointer && len(expectedFieldType.TypeParams) > 0 {
					innerExpected = &expectedFieldType.TypeParams[0]
				}
				fieldValue, err = t.transformShapeNodeWithExpectedType(field.Shape, innerExpected)
			} else if field.Assertion != nil {
				fieldValue, err = t.transformAssertionValue(field.Assertion, expectedFieldType)
			} else {
				fieldValue = goast.NewIdent("nil")
			}
			if err != nil {
				return nil, fmt.Errorf("failed to transform field value: %w", err)
			}

			// Pointer field logic
			if expectedFieldType != nil && expectedFieldType.Ident == ast.TypePointer {
				// If value is nil, emit nil
				if ident, ok := fieldValue.(*goast.Ident); ok && ident.Name == "nil" {
					// emit nil
				} else if unaryExpr, isPointer := fieldValue.(*goast.UnaryExpr); isPointer && unaryExpr.Op == token.AND {
					// Fix: unwrap &*User{...} to &User{...}
					if innerUnary, isStar := unaryExpr.X.(*goast.UnaryExpr); isStar && innerUnary.Op == token.MUL {
						if _, isStruct := innerUnary.X.(*goast.CompositeLit); isStruct {
							fieldValue = &goast.UnaryExpr{
								Op: token.AND,
								X:  innerUnary.X,
							}
						} else {
							// emit as-is
						}
					} else {
						// emit as-is
					}
				} else if _, isStruct := fieldValue.(*goast.CompositeLit); isStruct {
					// For struct literals in pointer fields, wrap them in &
					fieldValue = &goast.UnaryExpr{
						Op: token.AND,
						X:  fieldValue,
					}
				} else {
					fieldValue = &goast.UnaryExpr{
						Op: token.AND,
						X:  fieldValue,
					}
				}
			}
			fieldName := name
			if t.ExportReturnStructFields {
				fieldName = capitalizeFirst(name)
			}
			// Add debug log for the field value emission
			t.log.WithFields(map[string]interface{}{
				"fieldName":         fieldName,
				"expectedFieldType": expectedFieldType,
				"fieldValueType":    fmt.Sprintf("%T", fieldValue),
				"function":          "transformShapeNodeWithExpectedType",
			}).Debug("Emitting field value in struct literal")
			fields = append(fields, &goast.KeyValueExpr{
				Key:   goast.NewIdent(fieldName),
				Value: fieldValue,
			})
		}

		// If the expected type is a pointer, emit &Type{...}
		if expectedType != nil && expectedType.Ident == ast.TypePointer {
			return &goast.UnaryExpr{
				Op: token.AND,
				X: &goast.CompositeLit{
					Type: structType,
					Elts: fields,
				},
			}, nil
		}
		return &goast.CompositeLit{
			Type: structType,
			Elts: fields,
		}, nil
	}

	// Handle case where expectedType is nil but we have a structType from findExistingTypeForShape
	if structType != nil {
		// Build the struct literal fields without expected field types
		fields := []goast.Expr{}

		// Canonicalize field order for deterministic emission
		fieldNames := make([]string, 0, len(shape.Fields))
		for name := range shape.Fields {
			fieldNames = append(fieldNames, name)
		}
		sort.Strings(fieldNames)

		for _, name := range fieldNames {
			field := shape.Fields[name]
			var fieldValue goast.Expr
			var err error

			// Handle field.Node if present
			if field.Node != nil {
				switch n := field.Node.(type) {
				case *ast.ShapeNode:
					fieldValue, err = t.transformShapeNodeWithExpectedType(n, nil)
				case ast.ShapeNode:
					fieldValue, err = t.transformShapeNodeWithExpectedType(&n, nil)
				case ast.ExpressionNode:
					fieldValue, err = t.transformExpression(n)
				default:
					fieldValue = goast.NewIdent("nil") // fallback for unsupported node types
				}
			} else if field.Shape != nil {
				fieldValue, err = t.transformShapeNodeWithExpectedType(field.Shape, nil)
			} else if field.Assertion != nil {
				fieldValue, err = t.transformAssertionValue(field.Assertion, nil)
			} else {
				fieldValue = goast.NewIdent("nil")
			}
			if err != nil {
				return nil, fmt.Errorf("failed to transform field value: %w", err)
			}

			fieldName := name
			if t.ExportReturnStructFields {
				fieldName = capitalizeFirst(name)
			}
			fields = append(fields, &goast.KeyValueExpr{
				Key:   goast.NewIdent(fieldName),
				Value: fieldValue,
			})
		}

		return &goast.CompositeLit{
			Type: structType,
			Elts: fields,
		}, nil
	}

	return nil, fmt.Errorf("failed to transform shape node with expected type")
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
