// Package transformerts converts a Forst AST to TypeScript declaration files
package transformerts

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/typechecker"
	"strings"

	"github.com/sirupsen/logrus"
)

// TypeMapping maps Forst types to TypeScript types
type TypeMapping struct {
	builtinTypes map[string]string
	userTypes    map[string]string
}

// NewTypeMapping creates a new type mapping with built-in types
func NewTypeMapping() *TypeMapping {
	return &TypeMapping{
		builtinTypes: map[string]string{
			"String":  "string",
			"Int":     "number",
			"Int64":   "number",
			"Int32":   "number",
			"Float":   "number",
			"Float64": "number",
			"Float32": "number",
			"Bool":    "boolean",
			"Any":     "any",
		},
		userTypes: make(map[string]string),
	}
}

// AddUserType adds a user-defined type to the mapping
func (tm *TypeMapping) AddUserType(forstType, tsType string) {
	tm.userTypes[forstType] = tsType
}

// GetTypeScriptType returns the TypeScript type for a Forst type
func (tm *TypeMapping) GetTypeScriptType(forstType string) string {
	// Check user-defined types first
	if tsType, exists := tm.userTypes[forstType]; exists {
		return tsType
	}

	// Check built-in types
	if tsType, exists := tm.builtinTypes[forstType]; exists {
		return tsType
	}

	// Default to any for unknown types
	return "any"
}

// TypeScriptTransformer converts a Forst AST to TypeScript declaration files
type TypeScriptTransformer struct {
	TypeChecker *typechecker.TypeChecker
	Output      *TypeScriptOutput
	log         *logrus.Logger
	typeMapping *TypeMapping
}

// TypeScriptOutput holds the generated TypeScript declaration code
type TypeScriptOutput struct {
	PackageName string
	Types       []string
	Functions   []string
	Extensions  []string
}

// New creates a new TypeScriptTransformer
func New(tc *typechecker.TypeChecker, log *logrus.Logger) *TypeScriptTransformer {
	if log == nil {
		log = logrus.New()
		log.Warnf("No logger provided, using default logger")
	}
	return &TypeScriptTransformer{
		TypeChecker: tc,
		Output:      &TypeScriptOutput{},
		log:         log,
		typeMapping: NewTypeMapping(),
	}
}

// TransformForstFileToTypeScript converts a Forst AST to TypeScript declaration file
func (t *TypeScriptTransformer) TransformForstFileToTypeScript(nodes []ast.Node) (*TypeScriptOutput, error) {
	// Build type mapping first
	t.buildTypeMapping()

	// Process all definitions first to build user type mappings
	for _, def := range t.TypeChecker.Defs {
		switch def := def.(type) {
		case ast.TypeDefNode:
			t.log.WithFields(logrus.Fields{
				"typeDef":  def.GetIdent(),
				"function": "TransformForstFileToTypeScript",
			}).Debug("Processing type definition")
			tsType, err := t.transformTypeDef(def)
			if err != nil {
				return nil, fmt.Errorf("failed to transform type def %s: %w", def.GetIdent(), err)
			}
			t.Output.AddType(tsType)
		}
	}

	// Then process the rest of the nodes
	for _, node := range nodes {
		switch n := node.(type) {
		case ast.PackageNode:
			t.Output.SetPackageName(string(n.Ident.ID))
		case ast.FunctionNode:
			tsFunction, err := t.transformFunction(n)
			if err != nil {
				return nil, fmt.Errorf("failed to transform function %s: %w", n.GetIdent(), err)
			}
			t.Output.AddFunction(tsFunction)
		}
	}

	// Generate client extensions
	t.generateClientExtensions()

	t.log.WithFields(logrus.Fields{
		"function": "TransformForstFileToTypeScript",
		"types":    len(t.Output.Types),
		"funcs":    len(t.Output.Functions),
	}).Debug("Generated TypeScript declaration file")

	return t.Output, nil
}

// buildTypeMapping creates a mapping from Forst types to TypeScript types
func (t *TypeScriptTransformer) buildTypeMapping() {
	// User types will be added as we process type definitions
	t.log.WithFields(logrus.Fields{
		"function": "buildTypeMapping",
	}).Debug("Built type mapping")
}

// transformTypeDef converts a Forst type definition to TypeScript
func (t *TypeScriptTransformer) transformTypeDef(def ast.TypeDefNode) (string, error) {
	typeName := string(def.Ident)

	switch expr := def.Expr.(type) {
	case ast.TypeDefShapeExpr:
		// Add user type mapping
		tsType := fmt.Sprintf("I%s", typeName)
		t.typeMapping.AddUserType(typeName, tsType)

		return t.transformShapeToTypeScript(&expr.Shape, typeName)
	case ast.TypeDefAssertionExpr:
		// Add user type mapping
		tsType := fmt.Sprintf("I%s", typeName)
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
		tsType := t.typeMapping.GetTypeScriptType(string(field.Type.Ident))
		fields = append(fields, fmt.Sprintf("  %s: %s;", fieldName, tsType))
	}

	return fmt.Sprintf("export interface I%s {\n%s\n}", typeName, strings.Join(fields, "\n")), nil
}

// transformAssertionToTypeScript converts a Forst assertion to TypeScript
func (t *TypeScriptTransformer) transformAssertionToTypeScript(assertion *ast.AssertionNode, typeName string) (string, error) {
	if assertion == nil {
		return "", fmt.Errorf("assertion is nil")
	}

	baseType := "any"
	if assertion.BaseType != nil {
		baseType = t.typeMapping.GetTypeScriptType(string(*assertion.BaseType))
	}

	return fmt.Sprintf("export interface I%s extends %s {}", typeName, baseType), nil
}

// transformFunction converts a Forst function to TypeScript declaration
func (t *TypeScriptTransformer) transformFunction(fn ast.FunctionNode) (string, error) {
	// Generate TypeScript function signature
	params := []string{}

	for _, param := range fn.Params {
		paramType := param.GetType()
		tsType := t.typeMapping.GetTypeScriptType(string(paramType.Ident))
		params = append(params, fmt.Sprintf("%s: %s", param.GetIdent(), tsType))
	}

	// Determine return type - infer from Forst return types
	returnType := "void"
	if len(fn.ReturnTypes) > 0 {
		tsType := t.typeMapping.GetTypeScriptType(string(fn.ReturnTypes[0].Ident))
		returnType = tsType
	}

	// Generate function declaration with proper Promise return type
	funcName := string(fn.Ident.ID)
	paramStr := strings.Join(params, ", ")

	return fmt.Sprintf("export function %s(%s): Promise<%s>;",
		funcName, paramStr, returnType), nil
}

// generateClientExtensions generates extensions to the ForstClient
func (t *TypeScriptTransformer) generateClientExtensions() {
	extensions := []string{
		"declare module '@forst/sidecar' {",
		"  interface ForstClient {",
	}

	// Add function declarations to the client interface
	for _, function := range t.Output.Functions {
		funcDecl := strings.TrimPrefix(function, "export function ")
		extensions = append(extensions, fmt.Sprintf("    %s", funcDecl))
	}

	extensions = append(extensions,
		"  }",
		"}",
	)

	t.Output.Extensions = extensions
}

// AddType adds a type definition to the output
func (o *TypeScriptOutput) AddType(tsType string) {
	o.Types = append(o.Types, tsType)
}

// AddFunction adds a function definition to the output
func (o *TypeScriptOutput) AddFunction(tsFunction string) {
	o.Functions = append(o.Functions, tsFunction)
}

// SetPackageName sets the package name
func (o *TypeScriptOutput) SetPackageName(name string) {
	o.PackageName = name
}

// GenerateFile generates the complete TypeScript declaration file
func (o *TypeScriptOutput) GenerateFile() string {
	var lines []string

	// Add types
	if len(o.Types) > 0 {
		lines = append(lines, "// Type definitions", strings.Join(o.Types, "\n\n"), "")
	}

	// Add function declarations
	if len(o.Functions) > 0 {
		lines = append(lines, "// Function declarations", strings.Join(o.Functions, "\n\n"), "")
	}

	// Add client extensions
	if len(o.Extensions) > 0 {
		lines = append(lines, "// Client extensions", strings.Join(o.Extensions, "\n"), "")
	}

	return strings.Join(lines, "\n")
}
