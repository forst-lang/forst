package parser

import "forst/internal/ast"

// parseDerefAssignableExpr parses *x, **x, *xs[i], … when used as an assignment target.
func (p *Parser) parseDerefAssignableExpr() (ast.ExpressionNode, bool) {
	if p.current().Type != ast.TokenStar {
		return nil, false
	}
	depth := 0
	for p.current().Type == ast.TokenStar {
		p.advance()
		depth++
	}
	base := p.parseIdentifierPrimary()
	base = p.parseIndexSuffixChain(base, 0)
	vn, ok := base.(ast.ValueNode)
	if !ok {
		return nil, false
	}
	for i := 0; i < depth; i++ {
		vn = ast.DereferenceNode{Value: vn}
	}
	expr, ok := vn.(ast.ExpressionNode)
	if !ok {
		return nil, false
	}
	return expr, true
}
