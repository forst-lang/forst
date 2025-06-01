package parser

import "forst/internal/ast"

func (p *Parser) parseVarStatement() ast.AssignmentNode {
	p.advance() // Consume 'var' token

	// Parse the variable identifier
	ident := p.expect(ast.TokenIdentifier)

	// Check if the next token is a colon
	if p.current().Type == ast.TokenColon {
		p.advance() // Consume colon
		typeNode := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
		var expr ast.ExpressionNode
		// Check for optional initializer
		if p.current().Type == ast.TokenEquals {
			p.advance() // consume '='
			expr = p.parseExpression()
		}

		node := ast.AssignmentNode{
			LValues: []ast.VariableNode{
				{Ident: ast.Ident{ID: ast.Identifier(ident.Value)}},
			},
			ExplicitTypes: []*ast.TypeNode{&typeNode},
			IsShort:       false, // var declarations are never short assignments
		}

		if expr != nil {
			node.RValues = []ast.ExpressionNode{expr}
		}

		return node
	}

	// If no colon, assume the next token is the type
	typeNode := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
	var expr ast.ExpressionNode
	// Check for optional initializer
	if p.current().Type == ast.TokenEquals {
		p.advance() // consume '='
		expr = p.parseExpression()
	}

	node := ast.AssignmentNode{
		LValues: []ast.VariableNode{
			{Ident: ast.Ident{ID: ast.Identifier(ident.Value)}},
		},
		ExplicitTypes: []*ast.TypeNode{&typeNode},
		IsShort:       false, // var declarations are never short assignments
	}

	if expr != nil {
		node.RValues = []ast.ExpressionNode{expr}
	}

	return node
}
