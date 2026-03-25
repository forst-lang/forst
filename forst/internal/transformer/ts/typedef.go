package transformerts

import (
	"fmt"
	"forst/internal/ast"
	"strings"
)

// transformTypeDef converts a Forst type definition to TypeScript
func (t *TypeScriptTransformer) transformTypeDef(def ast.TypeDefNode) (string, error) {
	typeName := string(def.Ident)

	switch expr := def.Expr.(type) {
	case ast.TypeDefShapeExpr:
		// Add user type mapping - use clean name without I prefix for better UX
		tsType := typeName
		t.typeMapping.AddUserType(typeName, tsType)

		return t.transformShapeToTypeScript(&expr.Shape, typeName)
	case ast.TypeDefAssertionExpr:
		// Add user type mapping
		tsType := typeName
		t.typeMapping.AddUserType(typeName, tsType)

		return t.transformAssertionToTypeScript(expr.Assertion, typeName)
	default:
		return "", fmt.Errorf("unsupported type definition expression: %T", expr)
	}
}

// transformShapeToTypeScript converts a Forst shape to TypeScript interface
func (t *TypeScriptTransformer) transformShapeToTypeScript(shape *ast.ShapeNode, typeName string) (string, error) {
	var fields []string

	for fieldName, field := range shape.Fields {
		tsType, err := t.typeMapping.GetTypeScriptType(field.Type)
		if err != nil {
			return "", fmt.Errorf("failed to get TypeScript type for shape field %s: %w", fieldName, err)
		}
		fields = append(fields, fmt.Sprintf("  %s: %s;", fieldName, tsType))
	}

	return fmt.Sprintf("export interface %s {\n%s\n}", typeName, strings.Join(fields, "\n")), nil
}

// transformAssertionToTypeScript converts a Forst assertion to TypeScript
func (t *TypeScriptTransformer) transformAssertionToTypeScript(assertion *ast.AssertionNode, typeName string) (string, error) {
	if assertion == nil {
		return "", fmt.Errorf("assertion is nil")
	}

	baseType := "any"
	if assertion.BaseType != nil {
		baseTypeNode := ast.TypeNode{Ident: *assertion.BaseType}
		var err error
		baseType, err = t.typeMapping.GetTypeScriptType(&baseTypeNode)
		if err != nil {
			return "", fmt.Errorf("failed to get TypeScript type for assertion base type %s: %w", *assertion.BaseType, err)
		}
	}

	return fmt.Sprintf("export interface %s extends %s {}", typeName, baseType), nil
}
