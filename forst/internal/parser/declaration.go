package parser

import "forst/internal/ast"

// parseVarDeclaration parses a var declaration statement
func (p *Parser) parseVarDeclaration() ast.AssignmentNode {
	ident := p.expect(ast.TokenIdentifier)
	p.expect(ast.TokenColon)
	typeNode := p.parseType(TypeIdentOpts{})

	var value ast.ExpressionNode
	if p.current().Type == ast.TokenEquals {
		p.advance() // consume =
		value = p.parseExpression()
	}

	return ast.AssignmentNode{
		LValues: []ast.VariableNode{
			{
				Ident: ast.Ident{ID: ast.Identifier(ident.Value)},
			},
		},
		RValues: []ast.ExpressionNode{
			value,
		},
		ExplicitTypes: []*ast.TypeNode{&typeNode},
	}
}
