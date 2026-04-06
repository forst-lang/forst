package parser

import "forst/internal/ast"

func (p *Parser) parseAssignableExpr() ast.ExpressionNode {
	if p.current().Type != ast.TokenIdentifier {
		p.FailWithParseError(p.current(), "expected identifier")
	}
	base := p.parseIdentifierPrimary()
	return p.parseIndexSuffixChain(base, 0)
}

func (p *Parser) parseAssignment() ast.AssignmentNode {
	lhs := p.parseAssignableExpr()
	return p.finishAssignment(lhs)
}

func (p *Parser) finishAssignment(lhs ast.ExpressionNode) ast.AssignmentNode {
	if _, ok := lhs.(ast.IndexExpressionNode); ok && p.current().Type == ast.TokenColon {
		p.FailWithParseError(p.current(), "cannot use type annotation on indexed assignment")
	}

	var explicitType *ast.TypeNode
	if v, ok := lhs.(ast.VariableNode); ok && p.current().Type == ast.TokenColon {
		p.advance() // consume ':'
		typeNode := p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
		explicitType = &typeNode
		lhs = ast.VariableNode{
			Ident:        v.Ident,
			ExplicitType: typeNode,
		}
	}

	assignToken := p.current()
	if assignToken.Type != ast.TokenEquals && assignToken.Type != ast.TokenColonEquals {
		p.FailWithParseError(assignToken, "Expected assignment or short assignment operator")
	}
	if assignToken.Type == ast.TokenColonEquals {
		if _, ok := lhs.(ast.VariableNode); !ok {
			p.FailWithParseError(assignToken, "cannot use := on this expression")
		}
	} else {
		switch lhs.(type) {
		case ast.VariableNode, ast.IndexExpressionNode:
		default:
			p.FailWithParseError(assignToken, "left-hand side of assignment must be a variable or index expression")
		}
	}
	p.advance()

	expr := p.parseExpression()

	return ast.AssignmentNode{
		LValues:       []ast.ExpressionNode{lhs},
		RValues:       []ast.ExpressionNode{expr},
		ExplicitTypes: []*ast.TypeNode{explicitType},
		IsShort:       assignToken.Type == ast.TokenColonEquals,
	}
}

func (p *Parser) parseMultipleAssignment() ast.AssignmentNode {
	firstIdent := p.expect(ast.TokenIdentifier)
	p.expect(ast.TokenComma)
	secondIdent := p.expect(ast.TokenIdentifier)

	assignToken := p.current()
	if assignToken.Type != ast.TokenEquals && assignToken.Type != ast.TokenColonEquals {
		p.FailWithParseError(assignToken, "Expected assignment or short assignment operator")
	}
	p.advance()

	var exprs []ast.ExpressionNode
	for {
		exprs = append(exprs, p.parseExpression())
		if p.current().Type != ast.TokenComma {
			break
		}
		p.advance()
	}

	return ast.AssignmentNode{
		LValues: []ast.ExpressionNode{
			ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(firstIdent.Value)}},
			ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(secondIdent.Value)}},
		},
		RValues:       exprs,
		ExplicitTypes: []*ast.TypeNode{nil, nil},
		IsShort:       assignToken.Type == ast.TokenColonEquals,
	}
}
