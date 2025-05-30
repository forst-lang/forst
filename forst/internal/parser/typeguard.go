package parser

import (
	"forst/internal/ast"
)

// parseTypeGuard parses a type guard declaration
func (p *Parser) parseTypeGuard() *ast.TypeGuardNode {
	p.expect(ast.TokenIs)
	p.expect(ast.TokenLParen)

	// Parse parameters
	var params []ast.ParamNode
	for p.current().Type != ast.TokenRParen {
		param := p.parseParameter()
		params = append(params, param)

		if p.current().Type == ast.TokenComma {
			p.advance()
		}
	}
	p.expect(ast.TokenRParen)

	// Parse type guard name
	name := p.expect(ast.TokenIdentifier)

	// Parse body
	body := p.parseBlock(&BlockContext{AllowReturn: true})

	return &ast.TypeGuardNode{
		Ident:      ast.Identifier(name.Value),
		Parameters: params,
		Body:       body,
	}
}
