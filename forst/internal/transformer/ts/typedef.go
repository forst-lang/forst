package transformerts

import (
	"fmt"
	"forst/internal/ast"
	"strings"
)

// transformTypeDef converts a Forst type definition to TypeScript
func (t *TypeScriptTransformer) transformTypeDef(def ast.TypeDefNode) (string, error) {
	typeName := string(def.Ident)

	exportName := GeneratedTypeExport(typeName)
	switch expr := def.Expr.(type) {
	case ast.TypeDefShapeExpr:
		t.typeMapping.AddUserType(typeName, exportName)

		return t.transformShapeToTypeScript(&expr.Shape, exportName)
	case ast.TypeDefErrorExpr:
		t.typeMapping.AddUserType(typeName, exportName)
		return t.transformShapeToTypeScript(&expr.Payload, exportName)
	case ast.TypeDefAssertionExpr:
		t.typeMapping.AddUserType(typeName, exportName)

		return t.transformAssertionToTypeScript(expr.Assertion, exportName)
	case ast.TypeDefBinaryExpr:
		canon, err := t.TypeChecker.TypeDefExprToTypeNode(expr)
		if err != nil {
			return "", fmt.Errorf("binary typedef %s: %w", typeName, err)
		}
		ts, err := t.typeMapping.GetTypeScriptType(&canon)
		if err != nil {
			return "", err
		}
		t.typeMapping.AddUserType(typeName, ts)
		return fmt.Sprintf("export type %s = %s", exportName, ts), nil
	default:
		return "", fmt.Errorf("unsupported type definition expression: %T", expr)
	}
}

// transformShapeToTypeScript converts a Forst shape to TypeScript interface
func (t *TypeScriptTransformer) transformShapeToTypeScript(shape *ast.ShapeNode, typeName string) (string, error) {
	if shape == nil {
		return "", fmt.Errorf("shape is nil")
	}
	lines, err := t.typeMapping.shapeTypeFieldLines(*shape)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("export interface %s {\n%s\n}", typeName, strings.Join(lines, "\n")), nil
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
