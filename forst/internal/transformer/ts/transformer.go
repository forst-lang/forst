// Package transformerts converts a Forst AST to TypeScript declaration files
package transformerts

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// TypeScriptTransformer converts a Forst AST to TypeScript declaration files
type TypeScriptTransformer struct {
	TypeChecker *typechecker.TypeChecker
	Output      *TypeScriptOutput
	log         *logrus.Logger
	typeMapping *TypeMapping
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
