// Package transformergo converts a Forst AST to a Go AST
package transformergo

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
	goasttoken "go/token"

	"github.com/sirupsen/logrus"
)

// Transformer converts a Forst AST to a Go AST
type Transformer struct {
	TypeChecker          *typechecker.TypeChecker
	Output               *TransformerOutput
	assertionTransformer *AssertionTransformer
	log                  *logrus.Logger
}

// New creates a new Transformer
func New(tc *typechecker.TypeChecker, log *logrus.Logger) *Transformer {
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
	if scope.IsFunction() {
		return *scope.Node, nil
	}

	for scope != nil && !scope.IsFunction() && scope.Parent != nil {
		scope = scope.Parent
	}
	if scope.Node == nil {
		return ast.FunctionNode{}, fmt.Errorf("no function found")
	}
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
	t.log.WithFields(logrus.Fields{
		"function": "ensureAllReferencedTypesEmitted",
	}).Debug("Starting ensureAllReferencedTypesEmitted")

	// Collect all defined type names
	defined := map[string]bool{}
	for _, typeDecl := range t.Output.types {
		if len(typeDecl.Specs) > 0 {
			if spec, ok := typeDecl.Specs[0].(*goast.TypeSpec); ok {
				defined[spec.Name.Name] = true
			}
		}
	}

	// Collect all referenced type names from the output
	referenced := map[string]bool{}
	for _, typeDecl := range t.Output.types {
		if len(typeDecl.Specs) > 0 {
			if spec, ok := typeDecl.Specs[0].(*goast.TypeSpec); ok {
				// Scan the type definition for referenced types
				t.scanForReferencedTypes(spec.Type, referenced)
			}
		}
	}

	// Also scan function signatures and bodies for referenced types
	for _, funcDecl := range t.Output.functions {
		if funcDecl.Type != nil {
			// Scan function parameters
			if funcDecl.Type.Params != nil {
				for _, param := range funcDecl.Type.Params.List {
					if param.Type != nil {
						t.scanForReferencedTypes(param.Type, referenced)
					}
				}
			}
			// Scan function return types
			if funcDecl.Type.Results != nil {
				for _, result := range funcDecl.Type.Results.List {
					if result.Type != nil {
						t.scanForReferencedTypes(result.Type, referenced)
					}
				}
			}
		}
		// Scan function body for composite literals
		if funcDecl.Body != nil {
			t.scanFunctionBodyForReferencedTypes(funcDecl.Body, referenced)
		}
	}

	t.log.WithFields(logrus.Fields{
		"function":   "ensureAllReferencedTypesEmitted",
		"defined":    len(defined),
		"referenced": len(referenced),
	}).Debug("Collected type information")

	// Emit any missing referenced types
	for name := range referenced {
		if !defined[name] {
			t.log.WithFields(logrus.Fields{
				"function":    "ensureAllReferencedTypesEmitted",
				"missingType": name,
			}).Debug("Found missing referenced type")

			// Try to find the type definition in the type checker
			if def, ok := t.TypeChecker.Defs[ast.TypeIdent(name)]; ok {
				if typeDef, ok := def.(ast.TypeDefNode); ok {
					decl, err := t.transformTypeDef(typeDef)
					if err != nil {
						return err
					}
					if decl != nil {
						t.Output.AddType(decl)
						// Mark this type as defined to prevent duplicates
						defined[name] = true
					}
				}
			} else {
				// Try to synthesize a type definition from inferred types
				if synthesized := t.synthesizeTypeDefFromInferred(name); synthesized != nil {
					decl, err := t.transformTypeDef(*synthesized)
					if err != nil {
						return err
					}
					if decl != nil {
						t.Output.AddType(decl)
					}
				}
			}
		}
	}

	return nil
}

// scanForReferencedTypes recursively scans a Go AST expression for referenced type names
func (t *Transformer) scanForReferencedTypes(expr goast.Expr, referenced map[string]bool) {
	switch e := expr.(type) {
	case *goast.Ident:
		// Check if this looks like a hash-based type name (starts with T_)
		if len(e.Name) > 2 && e.Name[:2] == "T_" {
			referenced[e.Name] = true
		}
	case *goast.StarExpr:
		t.scanForReferencedTypes(e.X, referenced)
	case *goast.ArrayType:
		t.scanForReferencedTypes(e.Elt, referenced)
	case *goast.SelectorExpr:
		t.scanForReferencedTypes(e.X, referenced)
		t.scanForReferencedTypes(e.Sel, referenced)
	case *goast.CompositeLit:
		t.scanForReferencedTypes(e.Type, referenced)
		for _, elt := range e.Elts {
			if keyValue, ok := elt.(*goast.KeyValueExpr); ok {
				t.scanForReferencedTypes(keyValue.Key, referenced)
				t.scanForReferencedTypes(keyValue.Value, referenced)
			} else {
				t.scanForReferencedTypes(elt, referenced)
			}
		}
	}
}

// scanFunctionBodyForReferencedTypes scans a function body for composite literals that reference types
func (t *Transformer) scanFunctionBodyForReferencedTypes(body *goast.BlockStmt, referenced map[string]bool) {
	for _, stmt := range body.List {
		switch s := stmt.(type) {
		case *goast.AssignStmt:
			for _, rhs := range s.Rhs {
				t.scanForReferencedTypes(rhs, referenced)
			}
		case *goast.DeclStmt:
			if genDecl, ok := s.Decl.(*goast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*goast.ValueSpec); ok {
						for _, value := range valueSpec.Values {
							t.scanForReferencedTypes(value, referenced)
						}
					}
				}
			}
		case *goast.ExprStmt:
			t.scanForReferencedTypes(s.X, referenced)
		case *goast.ReturnStmt:
			for _, result := range s.Results {
				t.scanForReferencedTypes(result, referenced)
			}
		}
	}
}

// synthesizeTypeDefFromInferred attempts to synthesize a type definition from inferred types
func (t *Transformer) synthesizeTypeDefFromInferred(name string) *ast.TypeDefNode {
	// Look for the type in the type checker's inferred types
	for _, types := range t.TypeChecker.InferredTypes {
		if len(types) > 0 {
			// Check if any of the inferred types match our hash-based name
			for _, inferredType := range types {
				if inferredType.String() == name {
					// Found a match, synthesize a type definition
					// For now, just return nil since we can't easily synthesize from TypeNode
					// The type should already be defined in the type checker's Defs
					return nil
				}
			}
		}
	}
	return nil
}
