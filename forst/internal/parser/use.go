package parser

import (
	"strings"

	"forst/internal/ast"
)

func (p *Parser) parseUseStatement() ast.UseNode {
	startTok := p.current()
	p.advance() // past `use`

	if p.current().Value == "?" || strings.Contains(p.current().Value, "?") {
		p.FailWithParseError(p.current(), "optional provider use is not supported")
	}

	var binding *ast.Ident
	if p.current().Type == ast.TokenIdentifier && p.peek().Type == ast.TokenColon {
		nameTok := p.expect(ast.TokenIdentifier)
		if strings.Contains(nameTok.Value, "?") {
			p.FailWithParseError(nameTok, "optional provider use is not supported")
		}
		p.expect(ast.TokenColon)
		binding = &ast.Ident{
			ID:   ast.Identifier(nameTok.Value),
			Span: ast.SpanFromToken(nameTok),
		}
	}

	contractType := p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
	if strings.Contains(contractType.Ident.String(), "?") {
		p.FailWithParseError(p.tokens[p.currentIndex-1], "optional provider use is not supported")
	}
	endTok := p.tokens[p.currentIndex-1]

	node := ast.UseNode{
		Ident:        binding,
		ContractType: contractType,
		Span:         ast.SpanBetweenTokens(startTok, endTok),
	}
	return node
}
