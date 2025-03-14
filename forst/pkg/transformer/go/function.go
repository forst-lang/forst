package transformer_go

import (
	"forst/pkg/ast"
	goast "go/ast"
)

// transformFunction converts a Forst function node to a Go function declaration
func transformFunction(n ast.FunctionNode) *goast.FuncDecl {
	// Create function parameters
	params := &goast.FieldList{
		List: []*goast.Field{},
	}

	for _, param := range n.Params {
		params.List = append(params.List, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(param.Name)},
			Type:  goast.NewIdent(param.Type.Name),
		})
	}

	// Create function return type
	var results *goast.FieldList
	if !n.ReturnType.IsImplicit() {
		results = &goast.FieldList{
			List: []*goast.Field{
				{
					Type: goast.NewIdent(n.ReturnType.Name),
				},
			},
		}
	}

	// Create function body statements
	stmts := []goast.Stmt{}

	for _, stmt := range n.Body {
		stmts = append(stmts, transformStatement(stmt))
	}

	// Create the function declaration
	return &goast.FuncDecl{
		Name: goast.NewIdent(n.Name),
		Type: &goast.FuncType{
			Params:  params,
			Results: results,
		},
		Body: &goast.BlockStmt{
			List: stmts,
		},
	}
}
