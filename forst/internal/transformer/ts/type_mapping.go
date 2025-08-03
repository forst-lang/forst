package transformerts

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/typechecker"
	"strings"
)

// TypeMapping maps Forst types to TypeScript types
type TypeMapping struct {
	builtinTypes map[ast.TypeIdent]string
	userTypes    map[string]string
	typeChecker  *typechecker.TypeChecker
}

// NewTypeMapping creates a new type mapping with built-in types
func NewTypeMapping() *TypeMapping {
	return &TypeMapping{
		builtinTypes: map[ast.TypeIdent]string{
			ast.TypeString: "string",
			ast.TypeInt:    "number",
			ast.TypeFloat:  "number",
			ast.TypeBool:   "boolean",
			ast.TypeShape:  "object",
			ast.TypeVoid:   "void",
			ast.TypeObject: "object",
			ast.TypeArray:  "any[]",
			ast.TypeMap:    "Record<any, any>",
		},
		userTypes: make(map[string]string),
	}
}

// SetTypeChecker sets the typechecker for resolving hash-based types
func (tm *TypeMapping) SetTypeChecker(tc *typechecker.TypeChecker) {
	tm.typeChecker = tc
}

// AddUserType adds a user-defined type to the mapping
func (tm *TypeMapping) AddUserType(forstType, tsType string) {
	tm.userTypes[forstType] = tsType
}

// GetTypeScriptType returns the TypeScript type for a Forst type
func (tm *TypeMapping) GetTypeScriptType(forstType *ast.TypeNode) (string, error) {
	if forstType == nil {
		return "", fmt.Errorf("forstType required for GetTypeScriptType")
	}

	// Check user-defined types first
	if tsType, exists := tm.userTypes[string(forstType.Ident)]; exists {
		return tsType, nil
	}

	// Check built-in types
	if tsType, exists := tm.builtinTypes[forstType.Ident]; exists {
		return tsType, nil
	}

	// Handle hash-based types by resolving their underlying struct
	if forstType.TypeKind == ast.TypeKindHashBased && tm.typeChecker != nil {
		// Try to resolve the hash-based type to a named type
		aliasedName, err := tm.typeChecker.GetAliasedTypeName(*forstType, typechecker.GetAliasedTypeNameOptions{
			AllowStructuralAlias: true,
		})
		if err == nil && aliasedName != "" {
			// If we found an aliased name, use it
			return aliasedName, nil
		}

		// If no aliased name found, try to resolve the underlying struct definition
		if def, exists := tm.typeChecker.Defs[forstType.Ident]; exists {
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
					// Generate TypeScript interface from the shape
					return tm.generateTypeScriptInterface(shapeExpr.Shape), nil
				}
			}
		}
	}

	// Default to any for unknown types
	return "any", nil
}

// generateTypeScriptInterface generates a TypeScript interface from a shape
func (tm *TypeMapping) generateTypeScriptInterface(shape ast.ShapeNode) string {
	if len(shape.Fields) == 0 {
		return "{}"
	}

	var fields []string
	for fieldName, field := range shape.Fields {
		tsType, err := tm.GetTypeScriptType(field.Type)
		if err != nil {
			tsType = "any" // Fallback
		}
		fields = append(fields, fmt.Sprintf("  %s: %s;", fieldName, tsType))
	}

	return fmt.Sprintf("{\n%s\n}", strings.Join(fields, "\n"))
}
