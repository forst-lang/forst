package transformer_go

import (
	"fmt"
	"forst/pkg/ast"
	"forst/pkg/typechecker"
	goast "go/ast"
)

type Transformer struct {
	TypeChecker  *typechecker.TypeChecker
	currentScope *typechecker.Scope

	Output *TransformerOutput
}

func New(tc *typechecker.TypeChecker) *Transformer {
	return &Transformer{
		TypeChecker:  tc,
		currentScope: tc.GlobalScope(),
		Output:       &TransformerOutput{},
	}
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
				return nil, fmt.Errorf("failed to transform type def %s: %w", def.Ident, err)
			}
			t.Output.AddType(decl)
		}
	}

	for _, node := range nodes {
		switch n := node.(type) {
		case ast.PackageNode:
			t.Output.SetPackageName(string(n.Ident.Id))
		case ast.ImportNode:
			decl := t.transformImport(n)
			t.Output.AddImport(decl)
		case ast.ImportGroupNode:
			decl := t.transformImportGroup(n)
			t.Output.AddImportGroup(decl)
		case ast.FunctionNode:
			decl, err := t.transformFunction(n)
			if err != nil {
				return nil, fmt.Errorf("failed to transform function %s: %w", n.Ident.Id, err)
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
