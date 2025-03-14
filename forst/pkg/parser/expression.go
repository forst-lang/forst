package parser

import "forst/pkg/ast"

func (p *Parser) parseExpression() ast.ExpressionNode {
	return p.parseLiteral()
}
