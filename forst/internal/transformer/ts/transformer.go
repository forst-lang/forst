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

	// Handle hash-based types (generated types)
	if strings.HasPrefix(forstType, "T_") {
		return "any" // For now, use any for hash-based types
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

// TypeScriptOutput holds the generated TypeScript code
type TypeScriptOutput struct {
	PackageName string
	Types       []string
	Functions   []string
	ClientCode  []string // Per-package client code
	MainClient  string   // Main client class
	TypesFile   string   // Centralized types file
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

// TransformForstFileToTypeScript converts a Forst AST to TypeScript files
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

	// Generate the new client structure
	t.generateClientStructure()

	t.log.WithFields(logrus.Fields{
		"function": "TransformForstFileToTypeScript",
		"types":    len(t.Output.Types),
		"funcs":    len(t.Output.Functions),
	}).Debug("Generated TypeScript client structure")

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
		tsType := t.typeMapping.GetTypeScriptType(string(field.Type.Ident))
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
		baseType = t.typeMapping.GetTypeScriptType(string(*assertion.BaseType))
	}

	return fmt.Sprintf("export interface %s extends %s {}", typeName, baseType), nil
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
	returnType := "any"
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

// generateClientStructure generates the new Prisma-like client structure
func (t *TypeScriptTransformer) generateClientStructure() {
	if t.Output.PackageName == "" {
		return
	}

	// Generate types file
	t.generateTypesFile()

	// Generate per-package client code
	t.generatePackageClient()

	// Generate main client
	t.generateMainClient()
}

// generateTypesFile generates the centralized types file
func (t *TypeScriptTransformer) generateTypesFile() {
	var lines []string

	lines = append(lines, "// Auto-generated types for Forst client")
	lines = append(lines, "// Generated by Forst TypeScript Transformer")
	lines = append(lines, "")

	// Add all types
	if len(t.Output.Types) > 0 {
		lines = append(lines, "// Type definitions")
		lines = append(lines, strings.Join(t.Output.Types, "\n\n"))
		lines = append(lines, "")
	}

	// Add function signatures
	if len(t.Output.Functions) > 0 {
		lines = append(lines, "// Function signatures")
		lines = append(lines, strings.Join(t.Output.Functions, "\n\n"))
		lines = append(lines, "")
	}

	t.Output.TypesFile = strings.Join(lines, "\n")
}

// generatePackageClient generates the per-package client code
func (t *TypeScriptTransformer) generatePackageClient() {
	var lines []string

	lines = append(lines, fmt.Sprintf("// Auto-generated client for %s package", t.Output.PackageName))
	lines = append(lines, "// Generated by Forst TypeScript Transformer")
	lines = append(lines, "")
	lines = append(lines, "import { ForstClient as SidecarClient } from '@forst/sidecar';")
	lines = append(lines, "import * as types from './types';")
	lines = append(lines, "")

	// Generate package namespace
	lines = append(lines, fmt.Sprintf("export const %s = (client: SidecarClient) => ({", t.Output.PackageName))

	// Add function implementations
	for _, function := range t.Output.Functions {
		funcName := strings.TrimPrefix(function, "export function ")
		funcName = strings.Split(funcName, "(")[0]

		// Extract parameters
		paramPart := ""
		paramNames := []string{}
		if strings.Contains(function, "(") && strings.Contains(function, ")") {
			start := strings.Index(function, "(") + 1
			end := strings.Index(function, ")")
			paramPart = function[start:end]

			// Parse parameter names
			if paramPart != "" {
				params := strings.Split(paramPart, ", ")
				for _, param := range params {
					if strings.Contains(param, ":") {
						paramName := strings.Split(param, ":")[0]
						paramNames = append(paramNames, paramName)
					}
				}
			}
		}

		// Generate implementation
		impl := fmt.Sprintf("  %s: async (%s) => {", funcName, paramPart)

		// Create args object for single parameter, or use spread for multiple
		if len(paramNames) == 1 {
			impl += fmt.Sprintf("\n    const response = await client.invokeFunction('%s', '%s', %s);",
				t.Output.PackageName, funcName, paramNames[0])
		} else if len(paramNames) > 1 {
			impl += fmt.Sprintf("\n    const response = await client.invokeFunction('%s', '%s', { %s });",
				t.Output.PackageName, funcName, strings.Join(paramNames, ", "))
		} else {
			impl += fmt.Sprintf("\n    const response = await client.invokeFunction('%s', '%s', {});",
				t.Output.PackageName, funcName)
		}

		impl += "\n    if (!response.success) {"
		impl += fmt.Sprintf("\n      throw new Error(response.error || '%s.%s failed');",
			t.Output.PackageName, funcName)
		impl += "\n    }"
		impl += "\n    return response.result;"
		impl += "\n  },"

		lines = append(lines, impl)
	}

	lines = append(lines, "});")
	lines = append(lines, "")

	t.Output.ClientCode = lines
}

// generateMainClient generates the main client class
func (t *TypeScriptTransformer) generateMainClient() {
	var lines []string

	lines = append(lines, "// Auto-generated Forst Client")
	lines = append(lines, "// Generated by Forst TypeScript Transformer")
	lines = append(lines, "")
	lines = append(lines, "import { ForstClient as SidecarClient } from '@forst/sidecar';")
	lines = append(lines, fmt.Sprintf("import { %s } from './%s.client';", t.Output.PackageName, t.Output.PackageName))
	lines = append(lines, "import * as types from './types';")
	lines = append(lines, "")
	lines = append(lines, "export interface ForstClientConfig {")
	lines = append(lines, "  baseUrl?: string;")
	lines = append(lines, "  timeout?: number;")
	lines = append(lines, "  retries?: number;")
	lines = append(lines, "}")
	lines = append(lines, "")
	lines = append(lines, "export class ForstClient {")
	lines = append(lines, "  private client: SidecarClient;")
	lines = append(lines, fmt.Sprintf("  public %s: ReturnType<typeof %s>;", t.Output.PackageName, t.Output.PackageName))
	lines = append(lines, "")
	lines = append(lines, "  constructor(config?: ForstClientConfig) {")
	lines = append(lines, "    const defaultConfig = {")
	lines = append(lines, "      baseUrl: process.env.FORST_BASE_URL || 'http://localhost:8080',")
	lines = append(lines, "      timeout: 30000,")
	lines = append(lines, "      retries: 3,")
	lines = append(lines, "      ...config,")
	lines = append(lines, "    };")
	lines = append(lines, "")
	lines = append(lines, "    this.client = new SidecarClient(defaultConfig);")
	lines = append(lines, fmt.Sprintf("    this.%s = %s(this.client);", t.Output.PackageName, t.Output.PackageName))
	lines = append(lines, "  }")
	lines = append(lines, "}")
	lines = append(lines, "")
	lines = append(lines, "// Export types")
	lines = append(lines, "export * from './types';")
	lines = append(lines, "")
	lines = append(lines, "// Export default")
	lines = append(lines, "export default ForstClient;")

	t.Output.MainClient = strings.Join(lines, "\n")
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

// GenerateTypesFile returns the types file content
func (o *TypeScriptOutput) GenerateTypesFile() string {
	return o.TypesFile
}

// GenerateClientFile returns the package client file content
func (o *TypeScriptOutput) GenerateClientFile() string {
	return strings.Join(o.ClientCode, "\n")
}

// GenerateMainClient returns the main client file content
func (o *TypeScriptOutput) GenerateMainClient() string {
	return o.MainClient
}
