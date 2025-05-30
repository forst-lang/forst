package transformer_go

import (
	"forst/internal/ast"
	goast "go/ast"
)

func (t *Transformer) transformTypeGuard(n ast.TypeGuardNode) (*goast.FuncDecl, error) {
	// Create function parameters
	var params []*goast.Field

	// Transform subject parameter
	switch p := n.SubjectParam.(type) {
	case ast.SimpleParamNode:
		// Look up the type alias hash for the parameter's type
		var typeName string
		if t != nil {
			name, err := t.getTypeAliasNameForTypeNode(p.Type)
			if err != nil {
				typeName = string(p.Type.Ident)
			} else {
				typeName = name
			}
		} else {
			typeName = string(p.Type.Ident)
		}
		params = append(params, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(string(p.Ident.Id))},
			Type:  goast.NewIdent(typeName),
		})
	case ast.DestructuredParamNode:
		panic("DestructuredParamNode not supported in type guard")
	}

	// Transform additional parameters
	for _, param := range n.AdditionalParams {
		switch p := param.(type) {
		case ast.SimpleParamNode:
			var typeName string
			if t != nil {
				name, err := t.getTypeAliasNameForTypeNode(p.Type)
				if err != nil {
					typeName = string(p.Type.Ident)
				} else {
					typeName = name
				}
			} else {
				typeName = string(p.Type.Ident)
			}
			params = append(params, &goast.Field{
				Names: []*goast.Ident{goast.NewIdent(string(p.Ident.Id))},
				Type:  goast.NewIdent(typeName),
			})
		case ast.DestructuredParamNode:
			panic("DestructuredParamNode not supported in type guard")
		}
	}

	// Use hash-based guard function name
	hash := t.TypeChecker.Hasher.HashNode(n)
	guardFuncName := hash.ToGuardIdent()

	// Create function declaration
	return &goast.FuncDecl{
		Recv: nil,
		Name: goast.NewIdent(string(guardFuncName)),
		Type: &goast.FuncType{
			Params: &goast.FieldList{
				List: params,
			},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{
						Type: goast.NewIdent("bool"),
					},
				},
			},
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.ReturnStmt{
					Results: []goast.Expr{
						transformExpression(n.Body[0].(ast.ReturnNode).Value),
					},
				},
			},
		},
	}, nil
}
