package transformer_go

import (
	"forst/pkg/ast"
	goast "go/ast"
)

// transformFunction converts a Forst function node to a Go function declaration
func (t *Transformer) transformFunction(n ast.FunctionNode) (*goast.FuncDecl, error) {
	// Create function parameters
	params := &goast.FieldList{
		List: []*goast.Field{},
	}

	for _, param := range n.Params {
		params.List = append(params.List, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(param.Ident.String())},
			Type:  transformType(param.Type),
		})
	}

	// Create function return type
	returnType, err := t.TypeChecker.LookupFunctionReturnType(&n, t.currentScope)
	if err != nil {
		return nil, err
	}
	results := transformTypes(returnType)

	t.pushScope(n)

	// Create function body statements
	stmts := []goast.Stmt{}

	for _, stmt := range n.Body {
		stmts = append(stmts, t.transformStatement(stmt))
	}

	t.popScope()

	// Create the function declaration
	return &goast.FuncDecl{
		Name: goast.NewIdent(n.Ident.String()),
		Type: &goast.FuncType{
			Params:  params,
			Results: results,
		},
		Body: &goast.BlockStmt{
			List: stmts,
		},
	}, nil
}
