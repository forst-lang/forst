package main

import "fmt"

// Transforms Forst AST to Go AST
func TransformForstToGo(forstAST FuncNode) FuncNode {
	goAST := FuncNode{
		Name:       forstAST.Name,
		Params:     forstAST.Params,
		ReturnType: "string",
		Body:       []Node{},
	}

	for _, node := range forstAST.Body {
		switch n := node.(type) {
		case AssertNode:
			goAST.Body = append(goAST.Body, AssertNode{
				Condition: fmt.Sprintf("if !(%s) { return errors.New(\"%s\") }", n.Condition, n.ErrorType),
				ErrorType: n.ErrorType,
			})
		case ReturnNode:
			goAST.Body = append(goAST.Body, ReturnNode{
				Value: fmt.Sprintf(`return "%s"`, n.Value),
			})
		}
	}

	return goAST
}