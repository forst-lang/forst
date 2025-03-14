package transformer_go

import (
	"forst/pkg/ast"
	goast "go/ast"
)

// TransformForstToGo converts a Forst AST to a Go AST
func TransformForstFileToGo(nodes []ast.Node) *goast.File {
	var packageName string
	var decls []goast.Decl

	// Extract package name and declarations
	for _, node := range nodes {
		switch n := node.(type) {
		case ast.PackageNode:
			packageName = n.Value
		case ast.FunctionNode:
			decls = append(decls, transformFunction(n))
		}
	}

	if packageName == "" {
		packageName = "main"
	}

	// Create file node
	file := &goast.File{
		Name:  goast.NewIdent(packageName),
		Decls: decls,
	}

	return file
}
