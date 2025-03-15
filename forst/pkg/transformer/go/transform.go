package transformer_go

import (
	"fmt"
	"forst/pkg/ast"
	"forst/pkg/typechecker"
	goast "go/ast"
)

type Transformer struct{}

func New() *Transformer {
	return &Transformer{}
}

// TransformForstFileToGo converts a Forst AST to a Go AST
// The nodes should already have their types inferred/checked
func (t *Transformer) TransformForstFileToGo(nodes []ast.Node, tc *typechecker.TypeChecker) (*goast.File, error) {
	var packageName string
	var decls []goast.Decl

	for _, node := range nodes {
		switch n := node.(type) {
		case ast.PackageNode:
			packageName = n.Ident.Name
		case ast.FunctionNode:
			decl, err := t.transformFunction(n, tc)
			if err != nil {
				return nil, fmt.Errorf("failed to transform function %s: %w", n.Ident.Name, err)
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
