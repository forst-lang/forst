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
					inferredTypes, err := t.TypeChecker.InferAssertionType(param.Type.Assertion, false)
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
		// Try to use expected type if available (from assignment or function param)
		// For now, try to guess from context: if this is an assignment or argument, use the variable/param type
		// (This requires further integration with assignment/call transformation logic)
		return t.transformShapeNodeWithExpectedType(&e, nil)
	}

	return nil, fmt.Errorf("unsupported expression type: %s", reflect.TypeOf(expr).String())
}

// transformAssertionValue transforms an assertion value to a Go expression
func (t *Transformer) transformAssertionValue(assertion *ast.AssertionNode) (goast.Expr, error) {
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
					return &goast.UnaryExpr{
						Op: token.AND,
						X:  innerExpr,
					}, nil
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

// transformShapeNodeWithExpectedType generates a struct literal using the expected type if possible
func (t *Transformer) transformShapeNodeWithExpectedType(shape *ast.ShapeNode, expectedType *ast.TypeNode) (goast.Expr, error) {
	t.log.Debugf("transformShapeNodeWithExpectedType: shape with %d fields, expectedType=%v", len(shape.Fields), expectedType)

	var structType goast.Expr
	var fieldTypes map[string]string
	var isPointerType bool

	if expectedType == nil {
		// Try to get the hash-based type name for this shape
		hash, err := t.TypeChecker.Hasher.HashNode(*shape)
		if err == nil {
			expectedTypeName := string(hash.ToTypeIdent())
			t.log.Debugf("transformShapeNodeWithExpectedType: using hash-based type name %q", expectedTypeName)

			// Get the correct alias name for this type
			aliasName, err := t.getTypeAliasNameForTypeNode(ast.TypeNode{Ident: ast.TypeIdent(expectedTypeName)})
			if err != nil {
				t.log.Debugf("transformShapeNodeWithExpectedType: failed to get alias for %q: %v", expectedTypeName, err)
				// Fallback to original name
				aliasName = expectedTypeName
			}

			// Use the alias name as the composite literal type
			structType = goast.NewIdent(aliasName)
			t.log.Debugf("transformShapeNodeWithExpectedType: using struct type %q (alias of %q)", aliasName, expectedTypeName)
		}
	} else {
		// Check if this is a pointer type by examining the type structure
		if expectedType.Ident == ast.TypePointer {
			isPointerType = true
			t.log.Debugf("transformShapeNodeWithExpectedType: detected pointer type")

			// Get the base type (the type being pointed to) from TypeParams
			if len(expectedType.TypeParams) == 0 {
				return nil, fmt.Errorf("pointer type missing base type in TypeParams")
			}
			baseType := &expectedType.TypeParams[0]

			// Get the correct alias name for the base type
			aliasName, err := t.getTypeAliasNameForTypeNode(*baseType)
			if err != nil {
				t.log.Debugf("transformShapeNodeWithExpectedType: failed to get alias for base type: %v", err)
				// Fallback to original name
				aliasName = string(baseType.Ident)
			}

			// Use the base type as the composite literal type
			structType = goast.NewIdent(aliasName)
			t.log.Debugf("transformShapeNodeWithExpectedType: using base type %q for pointer", aliasName)
		} else {
			// Regular type (not a pointer)
			aliasName, err := t.getTypeAliasNameForTypeNode(*expectedType)
			if err != nil {
				t.log.Debugf("transformShapeNodeWithExpectedType: failed to get alias for %q: %v", expectedType.Ident, err)
				// Fallback to original name
				aliasName = string(expectedType.Ident)
			}

			// Use the alias name as the composite literal type
			structType = goast.NewIdent(aliasName)
			t.log.Debugf("transformShapeNodeWithExpectedType: using struct type %q", aliasName)
		}

		// Check if this type is actually defined in tc.Defs
		if expectedType.Ident == ast.TypePointer {
			// For pointer types, check if the base type exists
			if len(expectedType.TypeParams) > 0 {
				baseType := expectedType.TypeParams[0]
				if _, exists := t.TypeChecker.Defs[baseType.Ident]; exists {
					t.log.Debugf("transformShapeNodeWithExpectedType: found base type %q in tc.Defs", string(baseType.Ident))
				} else {
					t.log.Debugf("transformShapeNodeWithExpectedType: base type %q not found in tc.Defs, falling back to hash-based type", string(baseType.Ident))
					// Fall back to hash-based type name since the base type is not defined
					hash, err := t.TypeChecker.Hasher.HashNode(*shape)
					if err == nil {
						expectedTypeName := string(hash.ToTypeIdent())
						t.log.Debugf("transformShapeNodeWithExpectedType: using hash-based type name %q", expectedTypeName)

						// Get the correct alias name for this type
						aliasName, err := t.getTypeAliasNameForTypeNode(ast.TypeNode{Ident: ast.TypeIdent(expectedTypeName)})
						if err != nil {
							t.log.Debugf("transformShapeNodeWithExpectedType: failed to get alias for %q: %v", expectedTypeName, err)
							// Fallback to original name
							aliasName = expectedTypeName
						}

						// Use the alias name as the composite literal type
						structType = goast.NewIdent(aliasName)
						t.log.Debugf("transformShapeNodeWithExpectedType: using struct type %q (alias of %q)", aliasName, expectedTypeName)
					}
				}
			}
		} else {
			// For non-pointer types, check if the type exists
			if _, exists := t.TypeChecker.Defs[expectedType.Ident]; exists {
				t.log.Debugf("transformShapeNodeWithExpectedType: found type %q in tc.Defs", string(expectedType.Ident))
			} else {
				t.log.Debugf("transformShapeNodeWithExpectedType: type %q not found in tc.Defs, falling back to hash-based type", string(expectedType.Ident))
				// Fall back to hash-based type name since the expected type is not defined
				hash, err := t.TypeChecker.Hasher.HashNode(*shape)
				if err == nil {
					expectedTypeName := string(hash.ToTypeIdent())
					t.log.Debugf("transformShapeNodeWithExpectedType: using hash-based type name %q", expectedTypeName)

					// Get the correct alias name for this type
					aliasName, err := t.getTypeAliasNameForTypeNode(ast.TypeNode{Ident: ast.TypeIdent(expectedTypeName)})
					if err != nil {
						t.log.Debugf("transformShapeNodeWithExpectedType: failed to get alias for %q: %v", expectedTypeName, err)
						// Fallback to original name
						aliasName = expectedTypeName
					}

					// Use the alias name as the composite literal type
					structType = goast.NewIdent(aliasName)
					t.log.Debugf("transformShapeNodeWithExpectedType: using struct type %q (alias of %q)", aliasName, expectedTypeName)
				}
			}
		}

		// Check if this type is actually emitted in output
		outputTypeExists := false
		for _, decl := range t.Output.types {
			if typeSpec, ok := decl.Specs[0].(*goast.TypeSpec); ok {
				if typeSpec.Name.Name == structType.(*goast.Ident).Name {
					outputTypeExists = true
					t.log.Debugf("transformShapeNodeWithExpectedType: found type %q in output", structType.(*goast.Ident).Name)
					break
				}
			}
		}
		if !outputTypeExists {
			t.log.Debugf("transformShapeNodeWithExpectedType: WARNING - type %q not found in output!", structType.(*goast.Ident).Name)
		}

		// Optionally, try to extract field types from the struct if available
		for _, decl := range t.Output.types {
			if typeSpec, ok := decl.Specs[0].(*goast.TypeSpec); ok {
				if typeSpec.Name.Name == structType.(*goast.Ident).Name {
					if structTypeSpec, ok := typeSpec.Type.(*goast.StructType); ok {
						fieldTypes = make(map[string]string)
						for _, f := range structTypeSpec.Fields.List {
							if len(f.Names) > 0 {
								fieldTypes[f.Names[0].Name] = t.exprToTypeName(f.Type)
							}
						}
					}
					break
				}
			}
		}
	}

	if structType == nil {
		// Fallback: use anonymous struct type
		var err error
		structTypePtr, err := t.transformShapeType(shape)
		if err != nil {
			return nil, err
		}
		structType = *structTypePtr
	}

	// Build the struct literal fields, recursively using expected field types
	fields := []goast.Expr{}
	for name, field := range shape.Fields {
		var fieldValue goast.Expr
		var err error
		// If we have an expected field type, use it recursively
		var expectedFieldType *ast.TypeNode
		if fieldTypes != nil {
			fieldTypeName := fieldTypes[name]
			// Convert field type name back to TypeNode for recursive calls
			// This is a simplified approach - in a more complete implementation,
			// we'd want to preserve the full type structure
			if strings.HasPrefix(fieldTypeName, "*") {
				expectedFieldType = &ast.TypeNode{
					Ident:      ast.TypePointer,
					TypeParams: []ast.TypeNode{{Ident: ast.TypeIdent(fieldTypeName[1:])}},
				}
			} else {
				expectedFieldType = &ast.TypeNode{Ident: ast.TypeIdent(fieldTypeName)}
			}
		}
		if field.Shape != nil {
			fieldValue, err = t.transformShapeNodeWithExpectedType(field.Shape, expectedFieldType)
			if err != nil {
				return nil, err
			}
		} else if field.Assertion != nil {
			fieldValue, err = t.transformAssertionValue(field.Assertion)
			if err != nil {
				return nil, err
			}
		} else if field.Type != nil {
			// If the field has a type, use the zero value for that type
			goType, err := t.transformType(*field.Type)
			if err != nil {
				return nil, err
			}
			fieldValue = getZeroValue(goType)
		} else {
			fieldValue = goast.NewIdent("nil")
		}
		// Use original field name for Go struct literal key
		fieldName := name
		if t.ExportReturnStructFields {
			fieldName = capitalizeFirst(name)
		}
		fields = append(fields, &goast.KeyValueExpr{
			Key:   goast.NewIdent(fieldName),
			Value: fieldValue,
		})
	}

	// Create the struct literal
	structLiteral := &goast.CompositeLit{
		Type: structType,
		Elts: fields,
	}

	// If this is a pointer type, wrap the struct literal with an address-of operator
	if isPointerType {
		t.log.Debugf("transformShapeNodeWithExpectedType: wrapping struct literal with address-of operator")
		return &goast.UnaryExpr{
			Op: token.AND,
			X:  structLiteral,
		}, nil
	}

	return structLiteral, nil
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
