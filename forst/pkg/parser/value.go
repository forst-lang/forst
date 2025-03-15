package parser

import (
	"forst/pkg/ast"
)

func (p *Parser) parseValue(context *Context) ast.ValueNode {
	token := p.current()

	if token.Type == ast.TokenIdentifier {
		p.advance() // Consume identifier
		return ast.VariableNode{
			Ident: ast.Ident{Name: token.Value},
		}
	}

	return p.parseLiteral()
}
