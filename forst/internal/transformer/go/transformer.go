// Package transformergo converts a Forst AST to a Go AST
package transformergo

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
	goasttoken "go/token"

	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

// Transformer converts a Forst AST to a Go AST
type Transformer struct {
	TypeChecker          *typechecker.TypeChecker
	Output               *TransformerOutput
	assertionTransformer *AssertionTransformer
	log                  *logrus.Logger

	// If true, struct fields for return values will be exported (capitalized)
	ExportReturnStructFields bool
}

// New creates a new Transformer
func New(tc *typechecker.TypeChecker, log *logrus.Logger, exportReturnStructFields ...bool) *Transformer {
	if log == nil {
		log = logrus.New()
		log.Warnf("No logger provided, using default logger")
	}
	t := &Transformer{
		TypeChecker: tc,
		Output:      &TransformerOutput{},
		log:         log,
	}
	t.assertionTransformer = NewAssertionTransformer(t)
	if len(exportReturnStructFields) > 0 {
		t.ExportReturnStructFields = exportReturnStructFields[0]
	}
	return t
}

// TransformForstFileToGo converts a Forst AST to a Go AST
// The nodes should already have their types inferred/checked
func (t *Transformer) TransformForstFileToGo(nodes []ast.Node) (*goast.File, error) {
	// First, collect and register shape types from type definitions
	if err := t.defineShapeTypes(); err != nil {
		return nil, err
	}

	// Process all definitions first
	for _, def := range t.TypeChecker.Defs {
		switch def := def.(type) {
		case ast.TypeDefNode:
			t.log.WithFields(logrus.Fields{
				"typeDef":  def.GetIdent(),
				"function": "TransformForstFileToGo",
			}).Debug("Processing type definition")
			decl, err := t.transformTypeDef(def)
			if err != nil {
				return nil, fmt.Errorf("failed to transform type def %s: %w", def.GetIdent(), err)
			}
			t.Output.AddType(decl)
			t.log.WithFields(logrus.Fields{
				"typeDef":  def.GetIdent(),
				"function": "TransformForstFileToGo",
			}).Debug("Added type definition to output")
		case ast.TypeGuardNode:
			t.log.WithFields(logrus.Fields{
				"guard":    def.GetIdent(),
				"function": "TransformForstFileToGo",
			}).Debug("Processing type guard definition (value)")
			decl, err := t.transformTypeGuard(def)
			if err != nil {
				return nil, fmt.Errorf("failed to transform type guard %s: %w", def.GetIdent(), err)
			}
			if decl != nil {
				t.Output.AddFunction(decl)
			}
		case *ast.TypeGuardNode:
			t.log.WithFields(logrus.Fields{
				"guard":    def.GetIdent(),
				"function": "TransformForstFileToGo",
			}).Debug("Processing type guard definition (pointer)")
			decl, err := t.transformTypeGuard(*def)
			if err != nil {
				return nil, fmt.Errorf("failed to transform type guard %s: %w", def.GetIdent(), err)
			}
			if decl != nil {
				t.Output.AddFunction(decl)
			}
		case ast.TypeDefShapeExpr:
			decl, err := t.transformShapeType(&def.Shape)
			if err != nil {
				return nil, fmt.Errorf("failed to transform type def shape: %w", err)
			}
			t.Output.AddType(&goast.GenDecl{
				Tok: goasttoken.TYPE,
				Specs: []goast.Spec{
					&goast.TypeSpec{
						Name: goast.NewIdent(string(*def.Shape.BaseType)), // TODO: fix this
						Type: *decl,
					},
				},
			})
		}
	}

	// Then process the rest of the nodes
	for _, node := range nodes {
		switch n := node.(type) {
		case ast.PackageNode:
			t.Output.SetPackageName(string(n.Ident.ID))
		case ast.ImportNode:
			decl := t.transformImport(n)
			t.Output.AddImport(decl)
		case ast.ImportGroupNode:
			decl := t.transformImportGroup(n)
			t.Output.AddImportGroup(decl)
		case ast.FunctionNode:
			decl, err := t.transformFunction(n)
			if err != nil {
				return nil, fmt.Errorf("failed to transform function %s: %w", n.GetIdent(), err)
			}
			t.Output.AddFunction(decl)
		}
	}

	// Log the final output
	t.log.WithFields(logrus.Fields{
		"function": "TransformForstFileToGo",
		"types":    len(t.Output.types),
		"funcs":    len(t.Output.functions),
	}).Debug("Generated Go file")

	// Ensure all referenced types are emitted
	if err := t.ensureAllReferencedTypesEmitted(); err != nil {
		return nil, fmt.Errorf("failed to ensure all referenced types are emitted: %w", err)
	}

	return t.Output.GenerateFile()
}

func (t *Transformer) isMainPackage() bool {
	return t.Output.PackageName() == "main"
}

// closestFunction returns either the node corresponding to the current scope's function
// or, if the current scope is not a function, the next highest function node in the scope stack
// It returns an error if no function is found
func (t *Transformer) closestFunction() (ast.Node, error) {
	scope := t.currentScope()
	t.log.WithFields(map[string]interface{}{
		"scope":      scope,
		"isFunction": scope.IsFunction(),
		"function":   "closestFunction",
	}).Debug("Checking current scope")

	if scope.IsFunction() {
		t.log.WithFields(map[string]interface{}{
			"scope":    scope,
			"function": "closestFunction",
		}).Debug("Current scope is a function")
		return *scope.Node, nil
	}

	t.log.WithFields(map[string]interface{}{
		"scope":    scope,
		"function": "closestFunction",
	}).Debug("Current scope is not a function, searching up the scope stack")

	for scope != nil && !scope.IsFunction() && scope.Parent != nil {
		scope = scope.Parent
		t.log.WithFields(map[string]interface{}{
			"scope":      scope,
			"isFunction": scope.IsFunction(),
			"function":   "closestFunction",
		}).Debug("Checking parent scope")
	}
	if scope.Node == nil {
		t.log.WithFields(map[string]interface{}{
			"function": "closestFunction",
		}).Debug("No function found in scope stack")
		return ast.FunctionNode{}, fmt.Errorf("no function found")
	}

	t.log.WithFields(map[string]interface{}{
		"scope":    scope,
		"function": "closestFunction",
	}).Debug("Found function in scope stack")
	return (*scope.Node).(ast.FunctionNode), nil
}

func (t *Transformer) isMainFunction() bool {
	if !t.isMainPackage() {
		return false
	}

	scope := t.currentScope()
	if scope.IsGlobal() {
		t.log.Fatalf("isMainFunction called in global scope")
	}

	function, err := t.closestFunction()
	if err != nil {
		return false
	}
	if function, ok := function.(ast.FunctionNode); ok && function.HasMainFunctionName() {
		return true
	}

	return false
}

// ensureAllReferencedTypesEmitted ensures that all types referenced in the generated code are properly emitted
func (t *Transformer) ensureAllReferencedTypesEmitted() error {
	t.log.Debug("Starting ensureAllReferencedTypesEmitted")

	// Track which types we've already processed to avoid infinite recursion
	processed := make(map[ast.TypeIdent]bool)

	// First, recursively emit all referenced types from TypeChecker.Defs
	// Sort type definitions for deterministic emission order
	typeDefs := make([]struct {
		ident ast.TypeIdent
		def   ast.Node
	}, 0, len(t.TypeChecker.Defs))

	for typeIdent, def := range t.TypeChecker.Defs {
		typeDefs = append(typeDefs, struct {
			ident ast.TypeIdent
			def   ast.Node
		}{typeIdent, def})
	}

	// Sort by type identifier for deterministic order
	sort.Slice(typeDefs, func(i, j int) bool {
		return string(typeDefs[i].ident) < string(typeDefs[j].ident)
	})

	for _, typeDef := range typeDefs {
		if processed[typeDef.ident] {
			continue
		}
		processed[typeDef.ident] = true

		// Emit the type definition
		if err := t.emitTypeAndReferencedTypes(typeDef.ident, typeDef.def, processed); err != nil {
			return fmt.Errorf("failed to emit type definition %s: %w", typeDef.ident, err)
		}
	}

	// Then, recursively emit all referenced types from the generated code
	if err := t.scanAndEmitReferencedTypes(processed); err != nil {
		return fmt.Errorf("failed to scan and emit referenced types: %w", err)
	}

	return nil
}

// scanAndEmitReferencedTypes scans all generated code for referenced types and ensures they are emitted
func (t *Transformer) scanAndEmitReferencedTypes(processed map[ast.TypeIdent]bool) error {
	t.log.Debug("Scanning generated code for referenced types")

	// Scan all generated types for field types
	for _, typeDecl := range t.Output.types {
		if len(typeDecl.Specs) > 0 {
			if spec, ok := typeDecl.Specs[0].(*goast.TypeSpec); ok {
				if structType, ok := spec.Type.(*goast.StructType); ok {
					if structType.Fields != nil {
						for _, field := range structType.Fields.List {
							if field.Type != nil {
								if err := t.ensureTypeEmittedFromGoType(field.Type, processed); err != nil {
									return fmt.Errorf("failed to ensure field type emitted: %w", err)
								}
							}
						}
					}
				}
			}
		}
	}

	// Scan all generated functions for parameter and return types
	for _, funcDecl := range t.Output.functions {
		if funcDecl.Type != nil {
			// Scan parameter types
			if funcDecl.Type.Params != nil {
				for _, param := range funcDecl.Type.Params.List {
					if param.Type != nil {
						if err := t.ensureTypeEmittedFromGoType(param.Type, processed); err != nil {
							return fmt.Errorf("failed to ensure parameter type emitted: %w", err)
						}
					}
				}
			}
			// Scan return types
			if funcDecl.Type.Results != nil {
				for _, result := range funcDecl.Type.Results.List {
					if result.Type != nil {
						if err := t.ensureTypeEmittedFromGoType(result.Type, processed); err != nil {
							return fmt.Errorf("failed to ensure return type emitted: %w", err)
						}
					}
				}
			}
		}
	}

	return nil
}

// ensureTypeEmittedFromGoType ensures that a Go type is properly emitted if it represents a Forst type
func (t *Transformer) ensureTypeEmittedFromGoType(goType goast.Expr, processed map[ast.TypeIdent]bool) error {
	// Add debug log for the type being checked
	t.log.WithFields(logrus.Fields{
		"function": "ensureTypeEmittedFromGoType",
		"goType":   fmt.Sprintf("%#v", goType),
	}).Debug("[DEBUG] Checking type emission for goType")

	switch expr := goType.(type) {
	case *goast.Ident:
		// Check if this is a hash-based type name (starts with T_)
		if strings.HasPrefix(expr.Name, "T_") {
			typeIdent := ast.TypeIdent(expr.Name)
			if !processed[typeIdent] {
				t.log.WithFields(logrus.Fields{
					"function": "ensureTypeEmittedFromGoType",
					"type":     expr.Name,
				}).Debug("[DEBUG] Found hash-based type in generated code that needs emission")
				// Try to find this type in Defs
				if def, exists := t.TypeChecker.Defs[typeIdent]; exists {
					if err := t.emitTypeAndReferencedTypes(typeIdent, def, processed); err != nil {
						return fmt.Errorf("failed to emit referenced type %s: %w", typeIdent, err)
					}
				} else {
					t.log.WithFields(logrus.Fields{
						"function": "ensureTypeEmittedFromGoType",
						"type":     expr.Name,
					}).Warn("[DEBUG] Hash-based type found in generated code but not in Defs, creating minimal definition")
					// Create a minimal type definition to ensure emission
					minimalDef := ast.TypeDefNode{
						Ident: typeIdent,
						Expr: ast.TypeDefAssertionExpr{
							Assertion: &ast.AssertionNode{
								BaseType: func() *ast.TypeIdent { t := ast.TypeString; return &t }(),
								Constraints: []ast.ConstraintNode{{
									Name: "Value",
									Args: []ast.ConstraintArgumentNode{{
										Value: func() *ast.ValueNode {
											v := ast.ValueNode(ast.StringLiteralNode{Value: "placeholder"})
											return &v
										}(),
									}},
								}},
							},
						},
					}
					if err := t.emitTypeAndReferencedTypes(typeIdent, minimalDef, processed); err != nil {
						return fmt.Errorf("failed to emit minimal type definition for %s: %w", typeIdent, err)
					}
				}
			}
		}
	case *goast.StarExpr:
		// Handle pointer types recursively
		return t.ensureTypeEmittedFromGoType(expr.X, processed)
	case *goast.ArrayType:
		// Handle array types recursively
		return t.ensureTypeEmittedFromGoType(expr.Elt, processed)
	case *goast.MapType:
		// Handle map types recursively
		if err := t.ensureTypeEmittedFromGoType(expr.Key, processed); err != nil {
			return err
		}
		return t.ensureTypeEmittedFromGoType(expr.Value, processed)
	}
	return nil
}

// emitTypeAndReferencedTypes recursively emits a type and all types it references
func (t *Transformer) emitTypeAndReferencedTypes(typeIdent ast.TypeIdent, def interface{}, processed map[ast.TypeIdent]bool) error {
	// Add debug log for type emission
	t.log.WithFields(logrus.Fields{
		"function":  "emitTypeAndReferencedTypes",
		"typeIdent": typeIdent,
		"defType":   fmt.Sprintf("%T", def),
	}).Debug("[DEBUG] Emitting type and referenced types")
	// Skip if already processed
	if processed[typeIdent] {
		return nil
	}
	processed[typeIdent] = true

	// Check if already emitted
	alreadyEmitted := false
	for _, typeDecl := range t.Output.types {
		if len(typeDecl.Specs) > 0 {
			if spec, ok := typeDecl.Specs[0].(*goast.TypeSpec); ok {
				if spec.Name.Name == string(typeIdent) {
					alreadyEmitted = true
					break
				}
			}
		}
	}

	if alreadyEmitted {
		t.log.WithFields(logrus.Fields{
			"function":  "emitTypeAndReferencedTypes",
			"typeIdent": typeIdent,
		}).Debug("[DEBUG] Type already emitted, skipping")
		return nil
	}

	// Special case: emit type alias for hash-based types that are value constraints or primitive aliases
	if typeDef, ok := def.(ast.TypeDefNode); ok {
		if assertionExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok && assertionExpr.Assertion != nil {
			// If the assertion is a value constraint or base type is a primitive, emit alias
			if assertionExpr.Assertion.BaseType != nil {
				base := *assertionExpr.Assertion.BaseType
				t.log.WithFields(logrus.Fields{
					"function":  "emitTypeAndReferencedTypes",
					"typeIdent": typeIdent,
					"baseType":  base,
				}).Debug("[DEBUG] Emitting type alias for hash-based or primitive type")
				typeNode := ast.TypeNode{Ident: base}
				if typeNode.IsGoBuiltin() || base == ast.TypeString || base == ast.TypeInt || base == ast.TypeFloat || base == ast.TypeBool {
					goType, err := transformTypeIdent(base)
					if err == nil && goType != nil {
						t.Output.AddType(&goast.GenDecl{
							Tok: goasttoken.TYPE,
							Specs: []goast.Spec{
								&goast.TypeSpec{
									Name: goast.NewIdent(string(typeIdent)),
									Type: goType,
								},
							},
						})
						return nil
					}
				}
			}
		}
	}

	// Determine the type name to use for emission
	// For user-defined types (not hash-based), use the original name
	// For hash-based types, use the hash-based name
	typeNameToEmit := string(typeIdent)
	// The typeIdent is already the correct name to use - no need to change it
	// User-defined types will have their original names, hash-based types will have hash-based names

	// Transform and emit the type definition
	switch def := def.(type) {
	case ast.TypeDefNode:
		// First, emit all types referenced by this type definition
		if err := t.emitReferencedTypes(def, processed); err != nil {
			return fmt.Errorf("failed to emit referenced types for %s: %w", typeIdent, err)
		}

		// Then emit this type using the determined type name
		decl, err := t.transformTypeDef(def)
		if err != nil {
			t.log.WithFields(logrus.Fields{
				"function": "emitTypeAndReferencedTypes",
				"type":     string(typeIdent),
				"error":    err,
			}).Warn("Failed to transform type definition")
			return nil
		}
		if decl != nil {
			// Extract the type expression from the GenDecl
			var typeExpr goast.Expr
			if len(decl.Specs) > 0 {
				if spec, ok := decl.Specs[0].(*goast.TypeSpec); ok {
					typeExpr = spec.Type
				}
			}

			// Use the determined type name
			t.Output.AddType(&goast.GenDecl{
				Tok: goasttoken.TYPE,
				Specs: []goast.Spec{
					&goast.TypeSpec{
						Name: goast.NewIdent(typeNameToEmit),
						Type: typeExpr,
					},
				},
			})
			t.log.WithFields(logrus.Fields{
				"function":    "emitTypeAndReferencedTypes",
				"type":        string(typeIdent),
				"emittedName": typeNameToEmit,
			}).Debug("[DEBUG] Emitted type definition")
		}

	case ast.TypeDefShapeExpr:
		// First, emit all types referenced by this shape
		if err := t.emitReferencedTypesFromShape(&def.Shape, processed); err != nil {
			return fmt.Errorf("failed to emit referenced types for shape %s: %w", typeIdent, err)
		}

		// Then emit this shape type using the determined type name
		decl, err := t.transformShapeType(&def.Shape)
		if err != nil {
			t.log.WithFields(logrus.Fields{
				"function": "emitTypeAndReferencedTypes",
				"type":     string(typeIdent),
				"error":    err,
			}).Warn("Failed to transform shape type")
			return nil
		}
		if decl != nil {
			t.Output.AddType(&goast.GenDecl{
				Tok: goasttoken.TYPE,
				Specs: []goast.Spec{
					&goast.TypeSpec{
						Name: goast.NewIdent(typeNameToEmit),
						Type: *decl,
					},
				},
			})
			t.log.WithFields(logrus.Fields{
				"function":    "emitTypeAndReferencedTypes",
				"type":        string(typeIdent),
				"emittedName": typeNameToEmit,
			}).Debug("[DEBUG] Emitted shape type definition")
		}
	}

	return nil
}

// emitReferencedTypes emits all types referenced by a TypeDefNode
func (t *Transformer) emitReferencedTypes(def ast.TypeDefNode, processed map[ast.TypeIdent]bool) error {
	// Handle different expression types
	switch expr := def.Expr.(type) {
	case ast.TypeDefShapeExpr:
		return t.emitReferencedTypesFromShape(&expr.Shape, processed)
	case ast.TypeDefAssertionExpr:
		if expr.Assertion != nil {
			return t.emitReferencedTypesFromAssertion(expr.Assertion, processed)
		}
	}
	return nil
}

// emitReferencedTypesFromShape emits all types referenced by a shape
func (t *Transformer) emitReferencedTypesFromShape(shape *ast.ShapeNode, processed map[ast.TypeIdent]bool) error {
	for _, field := range shape.Fields {
		if field.Type != nil {
			// Emit the field type if it's a user-defined or hash-based type
			if !field.Type.IsGoBuiltin() {
				// Look up the type definition in TypeChecker.Defs
				if def, exists := t.TypeChecker.Defs[field.Type.Ident]; exists {
					if err := t.emitTypeAndReferencedTypes(field.Type.Ident, def, processed); err != nil {
						return fmt.Errorf("failed to emit field type %s: %w", field.Type.Ident, err)
					}
				}
			}
		}
		if field.Shape != nil {
			// Recursively emit nested shapes
			if err := t.emitReferencedTypesFromShape(field.Shape, processed); err != nil {
				return fmt.Errorf("failed to emit nested shape: %w", err)
			}
		}
		if field.Assertion != nil {
			// Emit types referenced by assertions
			if err := t.emitReferencedTypesFromAssertion(field.Assertion, processed); err != nil {
				return fmt.Errorf("failed to emit assertion type: %w", err)
			}
		}
	}
	return nil
}

// emitReferencedTypesFromAssertion emits all types referenced by an assertion
func (t *Transformer) emitReferencedTypesFromAssertion(assertion *ast.AssertionNode, processed map[ast.TypeIdent]bool) error {
	// Emit base type if it's user-defined
	if assertion.BaseType != nil {
		baseType := ast.TypeNode{Ident: *assertion.BaseType}
		if !baseType.IsGoBuiltin() {
			// Look up the type definition in TypeChecker.Defs
			if def, exists := t.TypeChecker.Defs[*assertion.BaseType]; exists {
				if err := t.emitTypeAndReferencedTypes(*assertion.BaseType, def, processed); err != nil {
					return fmt.Errorf("failed to emit assertion base type %s: %w", *assertion.BaseType, err)
				}
			}
		}
	}

	// Emit types referenced in constraints
	for _, constraint := range assertion.Constraints {
		for _, arg := range constraint.Args {
			if arg.Shape != nil {
				if err := t.emitReferencedTypesFromShape(arg.Shape, processed); err != nil {
					return fmt.Errorf("failed to emit constraint shape: %w", err)
				}
			}
		}
	}
	return nil
}

// getExpectedTypeForShape determines the expected type for a shape literal based on context.
// This function provides a unified way to determine the best type to use for struct literal emission.
// It prioritizes named types when available, falling back to hash-based types only when necessary.
func (t *Transformer) getExpectedTypeForShape(shape *ast.ShapeNode, context *ShapeContext) *ast.TypeNode {
	t.log.WithFields(logrus.Fields{
		"function": "getExpectedTypeForShape",
		"context":  fmt.Sprintf("%+v", context),
		"shape":    fmt.Sprintf("%+v", shape),
	}).Debug("[DEBUG] Determining expected type for shape literal")

	// If the shape has an explicit BaseType, use it
	if shape.BaseType != nil {
		t.log.WithFields(logrus.Fields{
			"function": "getExpectedTypeForShape",
			"baseType": *shape.BaseType,
		}).Debug("[DEBUG] Using explicit BaseType")
		return &ast.TypeNode{Ident: *shape.BaseType}
	}

	// If context provides an expected type, validate and use it
	if context != nil && context.ExpectedType != nil {
		expectedType := context.ExpectedType
		t.log.WithFields(logrus.Fields{
			"function":     "getExpectedTypeForShape",
			"expectedType": expectedType.Ident,
		}).Debug("[DEBUG] Context provided expected type")

		// Check if the expected type is compatible with the shape
		if def, exists := t.TypeChecker.Defs[expectedType.Ident]; exists {
			if typeDef, ok := def.(ast.TypeDefNode); ok {
				if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
					// Use typechecker to validate compatibility
					err := t.TypeChecker.ValidateShapeFields(shapeExpr.Shape, shape.Fields, expectedType.Ident)
					if err == nil {
						t.log.WithFields(logrus.Fields{
							"function":     "getExpectedTypeForShape",
							"expectedType": expectedType.Ident,
						}).Debug("[DEBUG] Expected type is compatible")
						return expectedType
					} else {
						t.log.WithFields(logrus.Fields{
							"function":     "getExpectedTypeForShape",
							"expectedType": expectedType.Ident,
							"error":        err.Error(),
						}).Debug("[DEBUG] Expected type is not compatible, will fall back to structural matching")
					}
				}
			}
		}
	}

	// Try to find a matching named type through structural matching
	typeIdent, found := t.findExistingTypeForShape(shape, nil)
	if found {
		t.log.WithFields(logrus.Fields{
			"function":  "getExpectedTypeForShape",
			"typeIdent": typeIdent,
		}).Debug("[DEBUG] Found matching named type through structural matching")
		return &ast.TypeNode{Ident: typeIdent}
	}

	// If context provides variable name, try to infer from variable type
	if context != nil && context.VariableName != "" {
		if types, ok := t.TypeChecker.VariableTypes[ast.Identifier(context.VariableName)]; ok && len(types) > 0 {
			expectedType := &types[0]
			t.log.WithFields(logrus.Fields{
				"function":     "getExpectedTypeForShape",
				"variableName": context.VariableName,
				"variableType": expectedType.Ident,
			}).Debug("[DEBUG] Found variable type for assignment")

			// Check if the variable type is compatible
			if def, exists := t.TypeChecker.Defs[expectedType.Ident]; exists {
				if typeDef, ok := def.(ast.TypeDefNode); ok {
					if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
						err := t.TypeChecker.ValidateShapeFields(shapeExpr.Shape, shape.Fields, expectedType.Ident)
						if err == nil {
							t.log.WithFields(logrus.Fields{
								"function":     "getExpectedTypeForShape",
								"variableType": expectedType.Ident,
							}).Debug("[DEBUG] Variable type is compatible")
							return expectedType
						}
					}
				}
			}
		}
	}

	// If context provides function name and parameter index, try to infer from function signature
	if context != nil && context.FunctionName != "" && context.ParameterIndex >= 0 {
		if sig, ok := t.TypeChecker.Functions[ast.Identifier(context.FunctionName)]; ok && context.ParameterIndex < len(sig.Parameters) {
			param := sig.Parameters[context.ParameterIndex]
			expectedType := &param.Type
			t.log.WithFields(logrus.Fields{
				"function":       "getExpectedTypeForShape",
				"functionName":   context.FunctionName,
				"parameterIndex": context.ParameterIndex,
				"parameterType":  expectedType.Ident,
			}).Debug("[DEBUG] Found function parameter type")

			// For assertion types, try to infer the concrete type
			if expectedType.Ident == ast.TypeAssertion && expectedType.Assertion != nil {
				inferredTypes, err := t.TypeChecker.InferAssertionType(expectedType.Assertion, false, "", nil)
				if err == nil && len(inferredTypes) > 0 {
					inferredType := &inferredTypes[0]
					t.log.WithFields(logrus.Fields{
						"function":     "getExpectedTypeForShape",
						"inferredType": inferredType.Ident,
					}).Debug("[DEBUG] Inferred concrete type from assertion")

					// Check if the inferred type is compatible
					if def, exists := t.TypeChecker.Defs[inferredType.Ident]; exists {
						if typeDef, ok := def.(ast.TypeDefNode); ok {
							if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
								err := t.TypeChecker.ValidateShapeFields(shapeExpr.Shape, shape.Fields, inferredType.Ident)
								if err == nil {
									t.log.WithFields(logrus.Fields{
										"function":     "getExpectedTypeForShape",
										"inferredType": inferredType.Ident,
									}).Debug("[DEBUG] Inferred type is compatible")
									return inferredType
								}
							}
						}
					}
				}
			} else {
				// For non-assertion types, check compatibility directly
				if def, exists := t.TypeChecker.Defs[expectedType.Ident]; exists {
					if typeDef, ok := def.(ast.TypeDefNode); ok {
						if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
							err := t.TypeChecker.ValidateShapeFields(shapeExpr.Shape, shape.Fields, expectedType.Ident)
							if err == nil {
								t.log.WithFields(logrus.Fields{
									"function":      "getExpectedTypeForShape",
									"parameterType": expectedType.Ident,
								}).Debug("[DEBUG] Parameter type is compatible")
								return expectedType
							}
						}
					}
				}
			}
		}
	}

	// If context provides return index, try to infer from function return type
	if context != nil && context.FunctionName != "" && context.ReturnIndex >= 0 {
		if sig, ok := t.TypeChecker.Functions[ast.Identifier(context.FunctionName)]; ok && context.ReturnIndex < len(sig.ReturnTypes) {
			returnType := &sig.ReturnTypes[context.ReturnIndex]
			t.log.WithFields(logrus.Fields{
				"function":     "getExpectedTypeForShape",
				"functionName": context.FunctionName,
				"returnIndex":  context.ReturnIndex,
				"returnType":   returnType.Ident,
			}).Debug("[DEBUG] Found function return type")

			// Check if the return type is compatible
			if def, exists := t.TypeChecker.Defs[returnType.Ident]; exists {
				if typeDef, ok := def.(ast.TypeDefNode); ok {
					if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
						err := t.TypeChecker.ValidateShapeFields(shapeExpr.Shape, shape.Fields, returnType.Ident)
						if err == nil {
							t.log.WithFields(logrus.Fields{
								"function":   "getExpectedTypeForShape",
								"returnType": returnType.Ident,
							}).Debug("[DEBUG] Return type is compatible")
							return returnType
						}
					}
				}
			}
		}
	}

	// No compatible named type found, return nil to indicate hash-based type should be used
	t.log.WithFields(logrus.Fields{
		"function": "getExpectedTypeForShape",
	}).Debug("[DEBUG] No compatible named type found, will use hash-based type")
	return nil
}

// ShapeContext provides context information for determining the expected type of a shape literal
type ShapeContext struct {
	// ExpectedType is the explicitly provided expected type
	ExpectedType *ast.TypeNode
	// VariableName is the name of the variable being assigned (for assignment context)
	VariableName string
	// FunctionName is the name of the function (for function call or return context)
	FunctionName string
	// ParameterIndex is the index of the parameter (for function call context)
	ParameterIndex int
	// ReturnIndex is the index of the return value (for return context)
	ReturnIndex int
}
