package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"

	logrus "github.com/sirupsen/logrus"
)

// getZeroValue returns the zero value for a Go type.
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
			return &goast.Ident{Name: "nil"}
		}
	case *goast.StarExpr:
		return &goast.Ident{Name: "nil"}
	case *goast.ArrayType:
		return &goast.Ident{Name: "nil"}
	case *goast.InterfaceType:
		return &goast.Ident{Name: "nil"}
	default:
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

// Helper: Build a composite literal of the expected type, mapping a variable to the first field, zero for others.
func (t *Transformer) buildCompositeLiteralForReturn(expectedType *ast.TypeNode, valueVar string) goast.Expr {
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
				val = goast.NewIdent("nil")
			}

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

// Helper: Build a zero composite literal of the expected type.
func (t *Transformer) buildZeroCompositeLiteral(expectedType *ast.TypeNode) goast.Expr {
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
			if field.Type.TypeKind == ast.TypeKindUserDefined {
				val = t.buildZeroCompositeLiteral(field.Type)
			} else {
				val = goast.NewIdent("nil")
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

// wrapVariableInNamedStruct generates a Go composite literal of the expected named struct type,
// mapping the variable to the appropriate field.
func (t *Transformer) wrapVariableInNamedStruct(expectedType *ast.TypeNode, variable ast.VariableNode) (goast.Expr, error) {
	goType, _ := t.transformType(*expectedType)

	def, ok := t.TypeChecker.Defs[expectedType.Ident].(ast.TypeDefNode)
	if !ok {
		return nil, fmt.Errorf("wrapVariableInNamedStruct: expected type %v is not a TypeDefNode", expectedType.Ident)
	}

	payload, ok := ast.PayloadShape(def.Expr)
	if !ok {
		return nil, fmt.Errorf("wrapVariableInNamedStruct: expected type %v does not have a shape or error payload", expectedType.Ident)
	}

	elts := make([]goast.Expr, 0, len(payload.Fields))
	for fieldName := range payload.Fields {
		var value goast.Expr

		fieldType := payload.Fields[fieldName].Type
		if fieldType != nil && fieldType.Ident == ast.TypeIdent(variable.Ident.ID) {
			value = goast.NewIdent(string(variable.Ident.ID))
		} else {
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
					value = goast.NewIdent("nil")
				}
			} else {
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
