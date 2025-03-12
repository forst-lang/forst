package transformers

import (
	"fmt"
	"forst/pkg/ast"
)

// Transforms Forst AST to Go AST
func TransformForstToGo(forstAST ast.FuncNode) ast.FuncNode {
	goAST := ast.FuncNode{
		Name:       forstAST.Name,
		Params:     forstAST.Params,
		ReturnType: "string",
		Body:       []ast.Node{},
	}

	for _, node := range forstAST.Body {
		switch n := node.(type) {
		case ast.AssertNode:
			goAST.Body = append(goAST.Body, ast.AssertNode{
				Condition: fmt.Sprintf("if !(%s) { return errors.New(\"%s\") }", n.Condition, n.ErrorType),
				ErrorType: n.ErrorType,
			})
		case ast.ReturnNode:
			goAST.Body = append(goAST.Body, ast.ReturnNode{
				Value: fmt.Sprintf(`return "%s"`, n.Value),
			})
		}
	}

	return goAST
}