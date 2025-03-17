package transformer_go

import (
	"fmt"
	"forst/pkg/ast"
	"forst/pkg/typechecker"
	goast "go/ast"
)

type Transformer struct {
	TypeChecker  *typechecker.TypeChecker
	packageName  string
	currentScope *typechecker.Scope
}

func New(tc *typechecker.TypeChecker) *Transformer {
	return &Transformer{
		TypeChecker:  tc,
		currentScope: tc.GlobalScope(),
	}
}

// TransformForstFileToGo converts a Forst AST to a Go AST
// The nodes should already have their types inferred/checked
func (t *Transformer) TransformForstFileToGo(nodes []ast.Node) (*goast.File, error) {
	var decls []goast.Decl

	// TODO: If we are calling ensure anywhere, import the errors package

	for _, node := range nodes {
		switch n := node.(type) {
		case ast.PackageNode:
			t.packageName = string(n.Ident.Id)
		case ast.ImportNode:
			decl := t.transformImport(n)
			decls = append(decls, decl)
		case ast.ImportGroupNode:
			decl := t.transformImportGroup(n)
			decls = append(decls, decl)
		case ast.FunctionNode:
			decl, err := t.transformFunction(n)
			if err != nil {
				return nil, fmt.Errorf("failed to transform function %s: %w", n.Ident.Id, err)
			}
			decls = append(decls, decl)
		}
	}

	if t.packageName == "nil" {
		t.packageName = "main"
	}

	file := &goast.File{
		Name:  goast.NewIdent(t.packageName),
		Decls: decls,
	}

	return file, nil
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

func (t *Transformer) isMainFunction() bool {
	if t.packageName != "main" {
		return false
	}

	currentFunction, err := t.currentFunction()
	if err != nil {
		return false
	}
	return currentFunction.Ident.Id == "main"
}
