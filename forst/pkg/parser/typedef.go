package parser

import "forst/pkg/ast"

func (p *Parser) parseTypeDef() ast.TypeDefNode {
	_ = p.expect(ast.TokenType)

	p.advance()

	name := p.expect(ast.TokenIdentifier).Value

	return ast.TypeDefNode{
		Name: name,
	}
}
