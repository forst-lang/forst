package transformerts

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/typechecker"
	"sort"
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
		// TypeArray / TypeMap use TypeParams; see switch in GetTypeScriptType.
		builtinTypes: map[ast.TypeIdent]string{
			ast.TypeString: "string",
			ast.TypeInt:    "number",
			ast.TypeFloat:  "number",
			ast.TypeBool:   "boolean",
			ast.TypeShape:  "object",
			ast.TypeVoid:   "void",
			ast.TypeObject: "object",
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

// sortedFieldNames returns shape field names in stable order for emitted TS.
func sortedFieldNames(fields map[string]ast.ShapeFieldNode) []string {
	if len(fields) == 0 {
		return nil
	}
	names := make([]string, 0, len(fields))
	for n := range fields {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// extractInlineShapeFromAssertion returns a nested shape from parser/typechecker
// encodings (Shape(...) or Match({ ... }) constraint with a shape argument).
func extractInlineShapeFromAssertion(a *ast.AssertionNode) *ast.ShapeNode {
	if a == nil {
		return nil
	}
	for _, c := range a.Constraints {
		if c.Name != "Shape" && c.Name != typechecker.ConstraintMatch {
			continue
		}
		for i := range c.Args {
			if c.Args[i].Shape != nil {
				return c.Args[i].Shape
			}
		}
	}
	return nil
}

// goBuiltinIdentToTypeScript maps Go primitive type idents (lowercase) to TS types.
func goBuiltinIdentToTypeScript(ident ast.TypeIdent) (string, bool) {
	if !typechecker.IsGoBuiltinType(string(ident)) {
		return "", false
	}
	switch string(ident) {
	case "string", "byte", "rune":
		return "string", true
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return "number", true
	case "bool":
		return "boolean", true
	case "error":
		return "unknown", true
	default:
		return "", false
	}
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

	if forstType.Ident == ast.TypeImplicit {
		return "unknown", nil
	}

	// Parameterized builtins (must run before the simple builtin map)
	switch forstType.Ident {
	case ast.TypeArray:
		if len(forstType.TypeParams) > 0 {
			elem, err := tm.GetTypeScriptType(&forstType.TypeParams[0])
			if err != nil {
				return "", err
			}
			return arrayTypeScript(elem), nil
		}
		return "any[]", nil
	case ast.TypeMap:
		if len(forstType.TypeParams) >= 2 {
			keyTS, err := tm.GetTypeScriptType(&forstType.TypeParams[0])
			if err != nil {
				return "", err
			}
			valTS, err := tm.GetTypeScriptType(&forstType.TypeParams[1])
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Record<%s, %s>", keyTS, valTS), nil
		}
		return "Record<any, any>", nil
	case ast.TypePointer:
		if len(forstType.TypeParams) > 0 {
			inner, err := tm.GetTypeScriptType(&forstType.TypeParams[0])
			if err != nil {
				return "", err
			}
			return "(" + inner + ") | null", nil
		}
		return "unknown", nil
	case ast.TypeError:
		return "unknown", nil
	case ast.TypeUnion:
		if len(forstType.TypeParams) == 0 {
			return "never", nil
		}
		parts := make([]string, 0, len(forstType.TypeParams))
		for i := range forstType.TypeParams {
			s, err := tm.GetTypeScriptType(&forstType.TypeParams[i])
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		}
		return strings.Join(parts, " | "), nil
	case ast.TypeIntersection:
		if len(forstType.TypeParams) == 0 {
			return "unknown", nil
		}
		parts := make([]string, 0, len(forstType.TypeParams))
		for i := range forstType.TypeParams {
			s, err := tm.GetTypeScriptType(&forstType.TypeParams[i])
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		}
		return strings.Join(parts, " & "), nil
	}

	// Inline object types from shape type definitions: { nested: { x: String } }
	if forstType.Ident == ast.TypeShape {
		if sh := extractInlineShapeFromAssertion(forstType.Assertion); sh != nil {
			return tm.shapeToInlineTypeScript(*sh)
		}
	}

	// Assertion types (e.g. User(Match({ ... })), refinements) — reuse typechecker inference.
	if forstType.Ident == ast.TypeAssertion {
		if tm.typeChecker != nil && forstType.Assertion != nil {
			inferred, err := tm.typeChecker.InferAssertionType(forstType.Assertion, false, "", nil)
			if err == nil && len(inferred) > 0 {
				return tm.GetTypeScriptType(&inferred[0])
			}
		}
		if sh := extractInlineShapeFromAssertion(forstType.Assertion); sh != nil {
			return tm.shapeToInlineTypeScript(*sh)
		}
		if forstType.Assertion != nil && forstType.Assertion.BaseType != nil {
			base := ast.TypeNode{
				Ident:    *forstType.Assertion.BaseType,
				TypeKind: ast.TypeKindUserDefined,
			}
			return tm.GetTypeScriptType(&base)
		}
		return "unknown", nil
	}

	// Go/TS interop: lowercase Go builtins appearing as TypeIdent strings
	if ts, ok := goBuiltinIdentToTypeScript(forstType.Ident); ok {
		return ts, nil
	}

	// Named type that exists in the typechecker (user-defined, stable name)
	if tm.typeChecker != nil && forstType.TypeKind == ast.TypeKindUserDefined && forstType.Ident != "" {
		if _, ok := tm.typeChecker.Defs[forstType.Ident].(ast.TypeDefNode); ok {
			return string(forstType.Ident), nil
		}
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
				if payload, ok := ast.PayloadShape(typeDef.Expr); ok {
					return tm.shapeToInlineTypeScript(*payload)
				}
			}
		}
	}

	if tsType, exists := tm.builtinTypes[forstType.Ident]; exists {
		return tsType, nil
	}

	// Default to any for unknown types
	return "any", nil
}

// arrayTypeScript returns a TS array type for an element type string (parenthesized when needed).
func arrayTypeScript(elem string) string {
	if strings.Contains(elem, "|") || strings.Contains(elem, " & ") {
		return "(" + elem + ")[]"
	}
	return elem + "[]"
}

// shapeFieldToTypeScript maps a single shape field using Type, nested Shape, or assertion inference.
func (tm *TypeMapping) shapeFieldToTypeScript(field ast.ShapeFieldNode) (string, error) {
	if field.Type != nil {
		return tm.GetTypeScriptType(field.Type)
	}
	if field.Shape != nil {
		return tm.shapeToInlineTypeScript(*field.Shape)
	}
	if field.Assertion != nil && tm.typeChecker != nil {
		inferred, err := tm.typeChecker.InferAssertionType(field.Assertion, false, "", nil)
		if err == nil && len(inferred) > 0 {
			return tm.GetTypeScriptType(&inferred[0])
		}
	}
	return "unknown", nil
}

// shapeTypeFieldLines returns sorted "  name: type;" lines for a shape (no surrounding braces).
func (tm *TypeMapping) shapeTypeFieldLines(shape ast.ShapeNode) ([]string, error) {
	names := sortedFieldNames(shape.Fields)
	if len(names) == 0 {
		return nil, nil
	}
	lines := make([]string, 0, len(names))
	for _, fieldName := range names {
		field := shape.Fields[fieldName]
		tsType, err := tm.shapeFieldToTypeScript(field)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", fieldName, err)
		}
		if tsType == "" {
			tsType = "any"
		}
		lines = append(lines, fmt.Sprintf("  %s: %s;", fieldName, tsType))
	}
	return lines, nil
}

// shapeToInlineTypeScript emits an inline object type for nested shapes and hash fallbacks.
func (tm *TypeMapping) shapeToInlineTypeScript(shape ast.ShapeNode) (string, error) {
	lines, err := tm.shapeTypeFieldLines(shape)
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "{}", nil
	}
	return fmt.Sprintf("{\n%s\n}", strings.Join(lines, "\n")), nil
}
