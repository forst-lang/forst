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
	var packageName string
	var decls []goast.Decl

	for _, node := range nodes {
		switch n := node.(type) {
		case ast.PackageNode:
			packageName = string(n.Ident.Id)
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

	if packageName == "" {
		packageName = "main"
	}

	file := &goast.File{
		Name:  goast.NewIdent(packageName),
		Decls: decls,
	}

	return file, nil
}
