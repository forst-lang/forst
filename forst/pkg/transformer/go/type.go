package transformer_go

import (
	"forst/pkg/ast"
	goast "go/ast"
)

// transformType converts a Forst type node to a Go type declaration
func transformType(n ast.TypeNode) *goast.Ident {
	switch n.Name {
	case ast.TypeInt:
		return goast.NewIdent("int")
	case ast.TypeFloat:
		return goast.NewIdent("float64")
	case ast.TypeString:
		return goast.NewIdent("string")
	case ast.TypeBool:
		return goast.NewIdent("bool")
	case ast.TypeVoid:
		return goast.NewIdent("void")
	}
	return goast.NewIdent(n.Name)
}
