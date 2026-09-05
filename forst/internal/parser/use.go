package parser

import (
	"strings"

	"forst/internal/ast"
)

func (p *Parser) parseUseStatement() ast.UseNode {
	startTok := p.current()
	p.advance() // past `use`

	if p.current().Value == "?" || strings.Contains(p.current().Value, "?") {
		p.FailWithReport(p.current(), "use-optional-unsupported", "optional provider use is not supported",
			"Optional provider bindings (`?`) are not supported.",
			"declare a required `use name: Type` binding instead")
	}

	var binding *ast.Ident
	if p.current().Type == ast.TokenIdentifier && p.peek().Type == ast.TokenColon {
		nameTok := p.expect(ast.TokenIdentifier)
		if strings.Contains(nameTok.Value, "?") {
			p.FailWithReport(nameTok, "use-optional-unsupported", "optional provider use is not supported",
				"Optional provider bindings (`?`) are not supported.",
				"declare a required `use name: Type` binding instead")
		}
		p.expect(ast.TokenColon)
		binding = &ast.Ident{
			ID:   ast.Identifier(nameTok.Value),
			Span: ast.SpanFromToken(nameTok),
		}
	}

	contractType := p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
	if strings.Contains(contractType.Ident.String(), "?") {
		p.FailWithReport(p.tokens[p.currentIndex-1], "use-optional-unsupported", "optional provider use is not supported",
			"Optional provider bindings (`?`) are not supported.",
			"declare a required `use name: Type` binding instead")
	}
	endTok := p.tokens[p.currentIndex-1]

	node := ast.UseNode{
		Ident:        binding,
		ContractType: contractType,
		Span:         ast.SpanBetweenTokens(startTok, endTok),
	}
	return node
}
