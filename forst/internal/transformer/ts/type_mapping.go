package transformerts

import (
	"fmt"
	"forst/internal/ast"
)

// TypeMapping maps Forst types to TypeScript types
type TypeMapping struct {
	builtinTypes map[ast.TypeIdent]string
	userTypes    map[string]string
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

	// Handle hash-based types (generated types)
	if forstType.TypeKind == ast.TypeKindHashBased {
		return "any", nil // For now, use any for hash-based types
	}

	// Default to any for unknown types
	return "any", nil
}
