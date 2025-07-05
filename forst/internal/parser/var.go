package parser

import (
	"forst/internal/ast"
)

func (p *Parser) parseVarStatement() ast.AssignmentNode {
	p.advance() // Consume 'var' token

	// Parse the variable identifier
	ident := p.expect(ast.TokenIdentifier)

	// Check for explicit type or assignment
	tok := p.current()

	if tok.Type == ast.TokenColon {
		// Explicit type: var x: Type = ... or var x: Type
		p.advance() // Consume colon
		typeNode := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})

		// Check if there's an equals sign for initialization
		if p.current().Type == ast.TokenEquals {
			p.expect(ast.TokenEquals)
			rvalue := p.parseExpression()
			return ast.AssignmentNode{
				LValues:       []ast.VariableNode{{Ident: ast.Ident{ID: ast.Identifier(ident.Value)}, ExplicitType: typeNode}},
				RValues:       []ast.ExpressionNode{rvalue},
				ExplicitTypes: []*ast.TypeNode{&typeNode},
			}
		} else {
			// No equals sign, just type declaration
			return ast.AssignmentNode{
				LValues:       []ast.VariableNode{{Ident: ast.Ident{ID: ast.Identifier(ident.Value)}, ExplicitType: typeNode}},
				ExplicitTypes: []*ast.TypeNode{&typeNode},
			}
		}
	} else if tok.Type == ast.TokenEquals {
		// No explicit type: var x = ...
		p.advance() // consume '='
		rvalue := p.parseExpression()
		if _, isNil := rvalue.(ast.NilLiteralNode); isNil {
			p.FailWithParseError(ident, "'var x = nil' is not allowed: explicit type required for nil assignment")
		}
		return ast.AssignmentNode{
			LValues: []ast.VariableNode{{Ident: ast.Ident{ID: ast.Identifier(ident.Value)}}},
			RValues: []ast.ExpressionNode{rvalue},
		}
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
			{
				Ident:        ast.Ident{ID: ast.Identifier(ident.Value)},
				ExplicitType: typeNode,
			},
		},
		ExplicitTypes: []*ast.TypeNode{&typeNode},
		IsShort:       false, // var declarations are never short assignments
	}

	if expr != nil {
		node.RValues = []ast.ExpressionNode{expr}
	}

	return node
}
