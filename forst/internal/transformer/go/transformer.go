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

	// Simple rule: emit all types that are explicitly defined in the type checker
	for typeIdent, def := range t.TypeChecker.Defs {
		// Skip if already emitted
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
			continue
		}

		// Transform and emit the type definition
		switch def := def.(type) {
		case ast.TypeDefNode:
			decl, err := t.transformTypeDef(def)
			if err != nil {
				t.log.WithFields(logrus.Fields{
					"function": "ensureAllReferencedTypesEmitted",
					"type":     string(typeIdent),
					"error":    err,
				}).Warn("Failed to transform type definition")
				continue
			}
			if decl != nil {
				t.Output.AddType(decl)
				t.log.WithFields(logrus.Fields{
					"function": "ensureAllReferencedTypesEmitted",
					"type":     string(typeIdent),
				}).Debug("Emitted type definition")
			}
		case ast.TypeDefShapeExpr:
			decl, err := t.transformShapeType(&def.Shape)
			if err != nil {
				t.log.WithFields(logrus.Fields{
					"function": "ensureAllReferencedTypesEmitted",
					"type":     string(typeIdent),
					"error":    err,
				}).Warn("Failed to transform shape type")
				continue
			}
			if decl != nil {
				t.Output.AddType(&goast.GenDecl{
					Tok: goasttoken.TYPE,
					Specs: []goast.Spec{
						&goast.TypeSpec{
							Name: goast.NewIdent(string(typeIdent)),
							Type: *decl,
						},
					},
				})
				t.log.WithFields(logrus.Fields{
					"function": "ensureAllReferencedTypesEmitted",
					"type":     string(typeIdent),
				}).Debug("Emitted shape type definition")
			}
		}
	}

	return nil
}
