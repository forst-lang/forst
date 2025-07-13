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

// determineStructType determines the Go struct type to use for a shape literal
func (t *Transformer) determineStructType(shape *ast.ShapeNode, expectedType *ast.TypeNode) (goast.Expr, error) {
	if expectedType != nil {
		t.log.WithFields(map[string]interface{}{
			"expectedType": expectedType.Ident,
			"function":     "determineStructType",
		}).Debug("[DEBUG] Using expected type directly for struct literal")
		return goast.NewIdent(string(expectedType.Ident)), nil
	}

	// Try to find an existing named type that matches this shape
	typeIdent, found := t.findExistingTypeForShape(shape, nil)
	t.log.WithFields(map[string]interface{}{
		"found":       found,
		"typeIdent":   typeIdent,
		"shapeFields": shape.Fields,
		"function":    "determineStructType",
	}).Warn("[DEBUG] findExistingTypeForShape result")
	if found {
		t.log.WithFields(map[string]interface{}{
			"typeIdent":   typeIdent,
			"shapeFields": shape.Fields,
			"function":    "determineStructType",
		}).Warn("[DEBUG] Found existing named type for shape without expectedType")
		return goast.NewIdent(string(typeIdent)), nil
	}

	t.log.WithFields(map[string]interface{}{
		"shapeFields": shape.Fields,
		"function":    "determineStructType",
	}).Warn("[DEBUG] No existing named type found for shape, will use hash-based type")

	// Fallback: Generate a hash-based type name
	hash, err := t.TypeChecker.Hasher.HashNode(shape)
	if err != nil {
		return nil, fmt.Errorf("failed to hash shape: %v", err)
	}
	typeName := hash.ToTypeIdent()
	t.log.WithFields(map[string]interface{}{
		"hash":     hash,
		"typeName": typeName,
		"function": "determineStructType",
	}).Warn("[DEBUG] Generated hash-based type name")

	return goast.NewIdent(string(typeName)), nil
}

// buildFieldsForExpectedType builds field expressions when an expected type is provided
func (t *Transformer) buildFieldsForExpectedType(shape *ast.ShapeNode, expectedType *ast.TypeNode) ([]*goast.KeyValueExpr, error) {
	fields := make([]*goast.KeyValueExpr, 0)

	def, ok := t.TypeChecker.Defs[expectedType.Ident]
	if !ok {
		return fields, nil
	}

	typeDef, ok := def.(ast.TypeDefNode)
	if !ok {
		return fields, nil
	}

	shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		return fields, nil
	}

	// Only emit fields that exist in the Go struct type, and fill missing ones with zero values
	for fieldName, fieldDef := range shapeExpr.Shape.Fields {
		var fieldValue goast.Expr
		var err error

		if field, ok := shape.Fields[fieldName]; ok {
			fieldValue, err = t.buildFieldValue(field, fieldDef)
		} else {
			// Field not present in literal, emit zero value
			fieldValue, err = t.buildZeroValue(fieldDef)
		}

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

// buildFieldsForShape builds field expressions when no expected type is provided
func (t *Transformer) buildFieldsForShape(shape *ast.ShapeNode) ([]*goast.KeyValueExpr, error) {
	fields := make([]*goast.KeyValueExpr, 0)

	for fieldName, field := range shape.Fields {
		fieldValue, err := t.buildFieldValue(field, ast.ShapeFieldNode{})
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

// buildFieldValue builds a field value expression
func (t *Transformer) buildFieldValue(field ast.ShapeFieldNode, fieldDef ast.ShapeFieldNode) (goast.Expr, error) {
	if field.Node != nil {
		if shapeNode, ok := field.Node.(ast.ShapeNode); ok {
			var fieldExpectedType *ast.TypeNode
			if fieldDef.Type != nil {
				fieldExpectedType = fieldDef.Type
			}
			return t.transformShapeNodeWithExpectedType(&shapeNode, fieldExpectedType)
		} else {
			return t.transformExpression(field.Node.(ast.ExpressionNode))
		}
	} else if field.Shape != nil {
		var fieldExpectedType *ast.TypeNode
		if fieldDef.Type != nil {
			fieldExpectedType = fieldDef.Type
		}
		return t.transformShapeNodeWithExpectedType(field.Shape, fieldExpectedType)
	} else if field.Assertion != nil {
		return t.transformAssertionValue(field.Assertion, nil)
	} else if field.Type != nil {
		return t.buildTypeValue(field.Type)
	}

	return goast.NewIdent("nil"), nil
}

// buildTypeValue builds a value for a type field
func (t *Transformer) buildTypeValue(fieldType *ast.TypeNode) (goast.Expr, error) {
	if fieldType.Ident == ast.TypePointer && len(fieldType.TypeParams) > 0 {
		return goast.NewIdent("nil"), nil
	}

	aliasName, err := t.TypeChecker.GetAliasedTypeName(*fieldType)
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
		} else {
			return &goast.CompositeLit{Type: goast.NewIdent(aliasName)}, nil
		}
	}
}

// buildZeroValue builds a zero value for a field definition
func (t *Transformer) buildZeroValue(fieldDef ast.ShapeFieldNode) (goast.Expr, error) {
	if fieldDef.Type == nil {
		return goast.NewIdent("nil"), nil
	}

	return t.buildTypeValue(fieldDef.Type)
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

	// Determine the struct type to use for the composite literal
	structType, err := t.determineStructType(shape, expectedType)
	if err != nil {
		return nil, err
	}

	t.log.WithFields(map[string]interface{}{
		"structType": fmt.Sprintf("%#v", structType),
		"function":   "transformShapeNodeWithExpectedType",
	}).Debug("[DEBUG] Final structType chosen for Go code generation")

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
