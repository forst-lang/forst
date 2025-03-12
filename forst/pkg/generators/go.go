package generators

import (
	"fmt"
	"forst/pkg/ast"
)

func GenerateGoCode(goAST ast.FuncNode) string {
	body := ""
	for _, node := range goAST.Body {
		switch n := node.(type) {
		case ast.AssertNode:
			body += fmt.Sprintf("    %s\n", n.Condition)
		case ast.ReturnNode:
			body += fmt.Sprintf("    %s\n", n.Value)
		}
	}

	return fmt.Sprintf(`package main

import (
	"fmt"
	"errors"
)

func %s() %s {
%s
}

func main() {
    fmt.Println(%s())
}
`, goAST.Name, goAST.ReturnType, body, goAST.Name)
}