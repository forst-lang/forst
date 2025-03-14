package generators

import (
	"bytes"
	goast "go/ast"
	"go/format"
	"go/token"
)

// GenerateGoCode generates Go code from a Go AST
func GenerateGoCode(funcDecl *goast.FuncDecl) string {
	// Create a file AST with the function
	file := &goast.File{
		Name: goast.NewIdent("main"),
		Decls: []goast.Decl{
			funcDecl,
			// Add main function
			&goast.FuncDecl{
				Name: goast.NewIdent("main"),
				Type: &goast.FuncType{
					Params: &goast.FieldList{},
				},
				Body: &goast.BlockStmt{
					List: []goast.Stmt{
						&goast.ExprStmt{
							X: &goast.CallExpr{
								Fun: &goast.SelectorExpr{
									X:   goast.NewIdent("fmt"),
									Sel: goast.NewIdent("Println"),
								},
								Args: []goast.Expr{
									&goast.CallExpr{
										Fun: goast.NewIdent(funcDecl.Name.Name),
									},
								},
							},
						},
					},
				},
			},
		},
		Imports: []*goast.ImportSpec{
			{
				Path: &goast.BasicLit{
					Kind:  token.STRING,
					Value: `"fmt"`,
				},
			},
			{
				Path: &goast.BasicLit{
					Kind:  token.STRING,
					Value: `"errors"`,
				},
			},
		},
	}

	// Format the AST
	var buf bytes.Buffer
	fset := token.NewFileSet()
	format.Node(&buf, fset, file)
	
	return buf.String()
}