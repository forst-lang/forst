package parser

import (
	"forst/pkg/ast"
)

func (p *Parser) parseValue() ast.ValueNode {
	token := p.current()

	if token.Type == ast.TokenIdentifier {
		p.advance() // Consume identifier
		return ast.VariableNode{
			Ident: ast.Ident{Id: ast.Identifier(token.Value)},
		}
	}

	return p.parseLiteral()
}
