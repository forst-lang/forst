package parser

import (
	"forst/internal/ast"
)

func (p *Parser) parseUseStatement() ast.UseNode {
	startTok := p.current()
	p.advance() // past `use`

	var binding *ast.Ident
	if p.current().Type == ast.TokenIdentifier && p.peek().Type == ast.TokenColon {
		nameTok := p.expect(ast.TokenIdentifier)
		p.expect(ast.TokenColon)
		binding = &ast.Ident{
			ID:   ast.Identifier(nameTok.Value),
			Span: ast.SpanFromToken(nameTok),
		}
	}

	contractType := p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
	endTok := p.tokens[p.currentIndex-1]

	node := ast.UseNode{
		Ident:        binding,
		ContractType: contractType,
		Span:         ast.SpanBetweenTokens(startTok, endTok),
	}
	return node
}
