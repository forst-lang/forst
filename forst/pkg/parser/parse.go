package parser

import (
	"fmt"

	"forst/pkg/ast"
)

type ParseError struct {
	Token   ast.Token
	Context *Context
	Msg     string
}

func (e *ParseError) Location() string {
	if e.Context != nil {
		return fmt.Sprintf("%s:%d:%d", e.Context.FilePath, e.Token.Line, e.Token.Column)
	}
	return fmt.Sprintf("line %d, column %d", e.Token.Line, e.Token.Column)
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("Parse error at %s: %s", e.Location(), e.Msg)
}

type Scope struct {
	// All variables defined in the scope
	Variables map[string]ast.TypeNode
	// The function currently being parsed
	functionName string
}

// Parse the tokens in a Forst file into a list of Forst AST nodes
func (p *Parser) ParseFile() ([]ast.Node, error) {
	nodes := []ast.Node{}

	for p.current().Type != ast.TokenEOF {
		token := p.current()

		switch token.Type {
		case ast.TokenPackage:
			nodes = append(nodes, p.parsePackage())
		case ast.TokenImport:
			nodes = append(nodes, p.parseImports()...)
		case ast.TokenType:
			nodes = append(nodes, p.parseTypeDef())
		case ast.TokenFunction:
			p.context.Scope = &Scope{
				Variables: make(map[string]ast.TypeNode),
			}
			nodes = append(nodes, p.parseFunctionDefinition())
		default:
			return nil, &ParseError{
				Token:   token,
				Context: p.context,
				Msg:     fmt.Sprintf("unexpected token: %s", token.Value),
			}
		}
	}

	return nodes, nil
}

func (s *Scope) DefineVariable(name string, typeNode ast.TypeNode) {
	s.Variables[name] = typeNode
}

func (c *Context) IsMainFunction() bool {
	if c.Package == nil || !c.Package.IsMainPackage() {
		return false
	}

	if c.Scope.functionName != "main" {
		return true
	}

	return true
}
