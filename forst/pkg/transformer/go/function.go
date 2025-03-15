package transformer_go

import (
	"forst/pkg/ast"
	"forst/pkg/typechecker"
	goast "go/ast"
)

// transformFunction converts a Forst function node to a Go function declaration
func (t *Transformer) transformFunction(n ast.FunctionNode, tc *typechecker.TypeChecker) (*goast.FuncDecl, error) {
	// Create function parameters
	params := &goast.FieldList{
		List: []*goast.Field{},
	}

	for _, param := range n.Params {
		params.List = append(params.List, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(param.Ident.Name)},
			Type:  transformType(param.Type),
		})
	}

	// Create function return type
	returnType, err := tc.LookupFunctionReturnType(&n)
	if err != nil {
		return nil, err
	}
	results := &goast.FieldList{
		List: []*goast.Field{
			{
				Type: transformType(returnType),
			},
		},
	}

	// Create function body statements
	stmts := []goast.Stmt{}

	for _, stmt := range n.Body {
		stmts = append(stmts, transformStatement(stmt))
	}

	// Create the function declaration
	return &goast.FuncDecl{
		Name: goast.NewIdent(n.Ident.Name),
		Type: &goast.FuncType{
			Params:  params,
			Results: results,
		},
		Body: &goast.BlockStmt{
			List: stmts,
		},
	}, nil
}
