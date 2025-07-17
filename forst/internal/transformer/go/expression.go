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
	// PINPOINT: Log before any return for type selection
	if expectedType != nil && !strings.HasPrefix(string(expectedType.Ident), "T_") {
		var typeName string

		// Handle pointer types properly
		if expectedType.Ident == ast.TypePointer && len(expectedType.TypeParams) > 0 {
			// For pointer types like *AppContext, extract the underlying type
			underlyingType := expectedType.TypeParams[0]
			if !strings.HasPrefix(string(underlyingType.Ident), "T_") {
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

		return goast.NewIdent(typeName), nil
	}

	// PINPOINT: Log when falling back to hash-based type
	if expectedType != nil {
		return goast.NewIdent(string(expectedType.Ident)), nil
	}

	// If expected type is provided, use it directly
	if expectedType != nil {
		// If the expected type is a pointer type, we need to preserve the base type
		// for proper code generation (e.g., Pointer(User) should generate &User{})
		if expectedType.Ident == ast.TypePointer && len(expectedType.TypeParams) > 0 {
			// For pointer types, we need to use the base type for the struct literal
			// but still indicate it's a pointer field
			baseType := expectedType.TypeParams[0]
			return goast.NewIdent(string(baseType.Ident)), nil
		}

		return goast.NewIdent(string(expectedType.Ident)), nil
	}

	// PINPOINT: When no expected type is provided, check if we should use a named type
	// instead of generating a hash-based type
	// Check if this shape matches any existing named type
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
	t.defineShapeType(shape)

	return goast.NewIdent(string(bestType.Ident)), nil
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
		// Resolve assertion fields to their actual types
		resolvedFieldDef := fieldDef
		if fieldDef.Assertion != nil && fieldDef.Type == nil {
			// For assertion fields like ctx:AppContext, resolve the assertion to get the actual type
			if assertionType, err := t.TypeChecker.InferAssertionType(fieldDef.Assertion, false, fieldName, nil); err == nil && len(assertionType) > 0 {
				// If the assertion refers to a named type, use that as the type ident
				resolvedTypeIdent := assertionType[0].Ident
				if fieldDef.Assertion.BaseType != nil && !strings.HasPrefix(string(resolvedTypeIdent), "T_") {
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

		fields = append(fields, &goast.KeyValueExpr{
			Key:   goast.NewIdent(fieldName),
			Value: fieldValue,
		})
	}

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
			if fieldDef.Type != nil {
				// For pointer types, preserve the pointer context for nested struct literals
				// This ensures they know they should be wrapped in &
				if fieldDef.Type.Ident == ast.TypePointer && len(fieldDef.Type.TypeParams) > 0 {
					// Pass the original pointer type to preserve context
					fieldExpectedType = fieldDef.Type
				} else if strings.HasPrefix(string(fieldDef.Type.Ident), "*") {
					// For pointer type identifiers like *User, pass the original pointer type
					fieldExpectedType = fieldDef.Type
				} else {
					// For non-pointer types, use the field type directly
					fieldExpectedType = fieldDef.Type
				}
			} else {
				// If fieldDef.Type is nil, use the expectedTypeForField passed from the parent
				fieldExpectedType = expectedTypeForField
			}
			value, err = t.transformShapeNodeWithExpectedType(&shapeNode, fieldExpectedType)
		} else {
			value, err = t.transformExpression(field.Node.(ast.ExpressionNode))
		}
	} else if field.Shape != nil {
		var fieldExpectedType *ast.TypeNode
		// Always use expectedTypeForField if it is a named type (not hash-based)
		if expectedTypeForField != nil && !strings.HasPrefix(string(expectedTypeForField.Ident), "T_") {
			fieldExpectedType = expectedTypeForField
		} else if fieldDef.Type != nil && !strings.HasPrefix(string(fieldDef.Type.Ident), "T_") {
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
		value = goast.NewIdent("nil")
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
			} else {
				// If no value provided, generate nil
				return goast.NewIdent("nil"), nil
			}
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
	// PINPOINT: Log entry with detailed context
	// If expectedType is a named type (not hash-based), always use it for the composite literal
	if expectedType != nil && !strings.HasPrefix(string(expectedType.Ident), "T_") {
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
				t.emitTypeAndReferencedTypes(ast.TypeIdent(ident.Name), def, processed)
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
	if expectedType != nil && !strings.HasPrefix(string(expectedType.Ident), "T_") {
		return *expectedType
	}

	// If the inferred type is already a named type, use it
	if !strings.HasPrefix(string(inferredType.Ident), "T_") {
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
