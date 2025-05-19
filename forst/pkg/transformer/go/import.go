package transformer_go

import (
	"forst/pkg/ast"
	goast "go/ast"
	"go/token"
	"strings"
)

func (t *Transformer) transformImport(node ast.ImportNode) *goast.GenDecl {
	return &goast.GenDecl{
		Tok: token.IMPORT,
		Specs: []goast.Spec{
			&goast.ImportSpec{
				Name: nameFromAlias(node.Alias),
				Path: &goast.BasicLit{
					Kind:  token.STRING,
					Value: `"` + strings.Trim(node.Path, `"`) + `"`,
				},
			},
		},
	}
}

func (t *Transformer) transformImportGroup(node ast.ImportGroupNode) *goast.GenDecl {
	specs := make([]goast.Spec, len(node.Imports))
	for i, imp := range node.Imports {
		specs[i] = &goast.ImportSpec{
			Name: nameFromAlias(imp.Alias),
			Path: &goast.BasicLit{
				Kind:  token.STRING,
				Value: `"` + strings.Trim(imp.Path, `"`) + `"`,
			},
		}
	}
	return &goast.GenDecl{
		Tok:   token.IMPORT,
		Specs: specs,
	}
}

func nameFromAlias(alias *ast.Ident) *goast.Ident {
	if alias == nil {
		return nil
	}
	return goast.NewIdent(string(alias.Id))
}
