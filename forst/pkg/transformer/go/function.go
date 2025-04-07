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
		var paramName string
		var paramType ast.TypeNode

		switch p := param.(type) {
		case ast.SimpleParamNode:
			paramName = string(p.Ident.Id)
			paramType = p.Type
		case ast.DestructuredParamNode:
			// Handle destructured params if needed
			continue
		}

		params.List = append(params.List, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(paramName)},
			Type:  t.transformType(paramType),
		})
	}

	// Create function return type
	returnType, err := t.TypeChecker.LookupFunctionReturnType(&n, t.currentScope)
	if err != nil {
		return nil, err
	}
	var results *goast.FieldList = nil
	isMainFunc := t.isMainPackage() && n.HasMainFunctionName()
	if !isMainFunc {
		results = t.transformTypes(returnType)
	}

	t.pushScope(n)

	// Create function body statements
	stmts := []goast.Stmt{}

	for _, stmt := range n.Body {
		goStmt := t.transformStatement(stmt)
		stmts = append(stmts, goStmt)
	}

	// Make sure that functions return nil if they return an error
	if !isMainFunc && len(returnType) > 0 {
		lastReturnType := returnType[len(returnType)-1]
		if lastReturnType.IsError() {
			var lastStmt ast.Node
			if len(n.Body) > 0 {
				lastStmt = n.Body[len(n.Body)-1]
			}
			if lastStmt == nil || lastStmt.Kind() != ast.NodeKindReturn {
				stmts = append(stmts, &goast.ReturnStmt{
					Results: []goast.Expr{
						goast.NewIdent("nil"),
					},
				})
			}
		}
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
