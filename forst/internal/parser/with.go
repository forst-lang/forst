package parser

import (
	"forst/internal/ast"
)

func (p *Parser) parseWithStatement() ast.WithNode {
	startTok := p.current()
	p.advance() // past `with`

	var wiring ast.ExpressionNode
	switch p.current().Type {
	case ast.TokenLBrace:
		shape := p.parseShapeLiteral(ShapeLiteralOpts{})
		wiring = shape
	default:
		if p.current().Type == ast.TokenIdentifier && p.peek().Type == ast.TokenLBrace {
			// Reject `with ctx { ... }` where ctx is a bare identifier (not a call or shape literal).
			p.FailWithReport(p.current(), "with-wiring", "with wiring must be a shape literal or call",
				"with wiring must be a shape literal `{ ... }` or a call expression, not a bare identifier.",
				"write `with { Logger: x }` or `with makeCtx() { ... }`")
		}
		wiring = p.parseExpression()
	}

	if p.current().Type != ast.TokenLBrace {
		p.FailWithReport(p.current(), "with-body", "with requires a body block",
			"with requires a body block `{ ... }` after the wiring expression.",
			"add `{ ... }` after the wiring expression")
	}

	body := p.parseBlock()
	endTok := p.tokens[p.currentIndex-1]

	return ast.WithNode{
		Wiring: wiring,
		Body:   body,
		Span:   ast.SpanBetweenTokens(startTok, endTok),
	}
}
