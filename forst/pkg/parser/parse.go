package parser

import (
	"fmt"

	"forst/pkg/ast"
)

type Scope struct {
	// All variables defined in the scope
	Variables map[string]ast.TypeNode
	// The name of the function currently being parsed
	FunctionName *string
}

// Mutable context for the parser to track the current state
type Context struct {
	IsMainPackage bool
	Scope         *Scope
}

const MAIN_FUNCTION_NAME = "main"

// Parse the tokens in a Forst file into a list of Forst AST nodes
func (p *Parser) ParseFile() []ast.Node {
	nodes := []ast.Node{}

	context := Context{
		IsMainPackage: false,
	}

	for p.current().Type != ast.TokenEOF {
		if p.current().Type == ast.TokenPackage {
			nodes = append(nodes, p.parsePackage(&context))
		} else if p.current().Type == ast.TokenImport {
			nodes = append(nodes, p.parseImports()...)
		} else if p.current().Type == ast.TokenFunction {
			context.Scope = &Scope{
				Variables: make(map[string]ast.TypeNode),
			}
			nodes = append(nodes, p.parseFunctionDefinition(&context))
		} else {
			panic(fmt.Sprintf("Unexpected token in file: %s", p.current().Value))
		}
	}

	return nodes
}

func (c *Context) IsMainFunction() bool {
	return c.IsMainPackage && c.Scope.FunctionName != nil && *c.Scope.FunctionName == MAIN_FUNCTION_NAME
}

func (s *Scope) DefineVariable(name string, typeNode ast.TypeNode) {
	s.Variables[name] = typeNode
}
