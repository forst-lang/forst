package parser

import "forst/internal/ast"

func (p *Parser) parseAssignment() ast.AssignmentNode {
	ident := p.expect(ast.TokenIdentifier)

	// Check for optional type annotation
	var explicitType *ast.TypeNode
	if p.current().Type == ast.TokenColon {
		p.advance() // consume ':'
		typeNode := p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
		explicitType = &typeNode
	}

	// Expect assignment operator
	assignToken := p.current()
	if assignToken.Type != ast.TokenEquals && assignToken.Type != ast.TokenColonEquals {
		p.FailWithParseError(assignToken, "Expected assignment or short assignment operator")
	}
	p.advance()

	expr := p.parseExpression()

	varIdent := ast.Ident{ID: ast.Identifier(ident.Value)}

	var lvalue ast.VariableNode
	if explicitType != nil {
		lvalue = ast.VariableNode{
			Ident:        varIdent,
			ExplicitType: *explicitType,
		}
	} else {
		lvalue = ast.VariableNode{
			Ident: varIdent,
		}
	}

	return ast.AssignmentNode{
		LValues:       []ast.VariableNode{lvalue},
		RValues:       []ast.ExpressionNode{expr},
		ExplicitTypes: []*ast.TypeNode{explicitType},
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
		p.FailWithParseError(assignToken, "Expected assignment or short assignment operator")
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
			{Ident: ast.Ident{ID: ast.Identifier(firstIdent.Value)}},
			{Ident: ast.Ident{ID: ast.Identifier(secondIdent.Value)}},
		},
		RValues:       exprs,
		ExplicitTypes: []*ast.TypeNode{nil, nil},
		IsShort:       assignToken.Type == ast.TokenColonEquals,
	}
}
