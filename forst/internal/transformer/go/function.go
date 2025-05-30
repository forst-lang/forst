package transformer_go

import (
	"fmt"
	"forst/internal/ast"
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
			paramName = string(p.Ident.ID)
			paramType = p.Type
		case ast.DestructuredParamNode:
			// Handle destructured params if needed
			continue
		}

		var ident *goast.Ident
		if t != nil {
			name, err := t.getTypeAliasNameForTypeNode(paramType)
			if err != nil {
				return nil, fmt.Errorf("failed to get type alias name: %s", err)
			}
			ident = goast.NewIdent(name)
		} else {
			ident = goast.NewIdent(string(paramType.Ident))
		}
		params.List = append(params.List, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(paramName)},
			Type:  ident,
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
		results, err = t.transformTypes(returnType)
		if err != nil {
			return nil, fmt.Errorf("failed to transform types: %s", err)
		}
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
