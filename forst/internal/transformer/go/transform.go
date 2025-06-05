// Package transformergo converts a Forst AST to a Go AST
package transformergo

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
)

// Transformer converts a Forst AST to a Go AST
type Transformer struct {
	TypeChecker          *typechecker.TypeChecker
	currentScope         *typechecker.Scope
	Output               *TransformerOutput
	assertionTransformer *AssertionTransformer
}

// New creates a new Transformer
func New(tc *typechecker.TypeChecker) *Transformer {
	t := &Transformer{
		TypeChecker:  tc,
		currentScope: tc.GlobalScope(),
		Output:       &TransformerOutput{},
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

	for _, def := range t.TypeChecker.Defs {
		switch def := def.(type) {
		case ast.TypeDefNode:
			decl, err := t.transformTypeDef(def)
			if err != nil {
				return nil, fmt.Errorf("failed to transform type def %s: %w", def.GetIdent(), err)
			}
			t.Output.AddType(decl)
		case ast.TypeGuardNode:
			decl, err := t.transformTypeGuard(def)
			if err != nil {
				return nil, fmt.Errorf("failed to transform type guard %s: %w", def.GetIdent(), err)
			}
			t.Output.AddFunction(decl)
		}
	}

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

	return t.Output.GenerateFile()
}

func (t *Transformer) currentFunction() (ast.FunctionNode, error) {
	scope := t.currentScope
	for scope != nil && !scope.IsFunction() && scope.Parent != nil {
		scope = scope.Parent
	}
	if scope.Node == nil {
		return ast.FunctionNode{}, fmt.Errorf("no function found")
	}
	return scope.Node.(ast.FunctionNode), nil
}

func (t *Transformer) isMainPackage() bool {
	return t.Output.PackageName() == "main"
}

func (t *Transformer) isMainFunction() bool {
	if !t.isMainPackage() {
		return false
	}

	currentFunction, err := t.currentFunction()
	if err != nil {
		return false
	}
	return currentFunction.HasMainFunctionName()
}
