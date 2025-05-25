package parser

import (
	"forst/internal/ast"
)

func (p *Parser) parseValue() ast.ValueNode {
	token := p.current()

	if token.Type == ast.TokenIdentifier {
		p.advance() // Consume identifier
		var identifier ast.Identifier = ast.Identifier(token.Value)

		// Keep chaining identifiers with dots until we hit something else
		for p.current().Type == ast.TokenDot {
			p.advance() // Consume dot
			nextIdent := p.expect(ast.TokenIdentifier)
			identifier = ast.Identifier(string(identifier) + "." + nextIdent.Value)
		}

		return ast.VariableNode{
			Ident: ast.Ident{Id: identifier},
		}
	}

	return p.parseLiteral()
}
