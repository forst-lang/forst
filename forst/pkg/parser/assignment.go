package parser

import "forst/pkg/ast"

func (p *Parser) parseAssignment() ast.AssignmentNode {
	ident := p.expect(ast.TokenIdentifier)
	// Expect assignment operator
	assignToken := p.current()
	if assignToken.Type != ast.TokenEquals && assignToken.Type != ast.TokenColonEquals {
		panic(parseErrorWithValue(assignToken, "Expected assignment or short assignment operator"))
	}
	p.advance()

	expr := p.parseExpression()

	return ast.AssignmentNode{
		LValues: []ast.VariableNode{
			{Ident: ast.Ident{Id: ast.Identifier(ident.Value)}},
		},
		RValues:       []ast.ExpressionNode{expr},
		ExplicitTypes: []*ast.TypeNode{nil},
		IsShort:       assignToken.Type == ast.TokenColonEquals,
	}
}

func (p *Parser) parseMultipleAssignment() ast.AssignmentNode {
	firstIdent := p.expect(ast.TokenIdentifier)
	p.expect(ast.TokenComma)
	secondIdent := p.expect(ast.TokenIdentifier)

	// Expect assignment operator
	assignToken := p.current()
	if assignToken.Type != ast.TokenEquals && assignToken.Type != ast.TokenColonEquals {
		panic(parseErrorWithValue(assignToken, "Expected assignment or short assignment operator"))
	}
	p.advance()

	// Parse comma-separated expressions
	var exprs []ast.ExpressionNode
	for {
		exprs = append(exprs, p.parseExpression())
		if p.current().Type != ast.TokenComma {
			break
		}
		p.advance() // Skip comma
	}

	return ast.AssignmentNode{
		LValues: []ast.VariableNode{
			{Ident: ast.Ident{Id: ast.Identifier(firstIdent.Value)}},
			{Ident: ast.Ident{Id: ast.Identifier(secondIdent.Value)}},
		},
		RValues:       exprs,
		ExplicitTypes: []*ast.TypeNode{nil, nil},
		IsShort:       assignToken.Type == ast.TokenColonEquals,
	}
}
