package parser

import (
	"fmt"
	"forst/pkg/ast"

	log "github.com/sirupsen/logrus"
)

func (p *Parser) parseShape() ast.ShapeNode {
	p.expect(ast.TokenLBrace)

	fields := make(map[string]ast.ShapeFieldNode)
	// Parse fields until closing brace
	for p.current().Type != ast.TokenRBrace {
		// Parse field name
		name := p.expect(ast.TokenIdentifier).Value
		p.expect(ast.TokenColon)

		// Parse field value (can be another shape or a type assertion)
		var value ast.ShapeFieldNode
		if p.current().Type == ast.TokenLBrace {
			shape := p.parseShape()
			value = ast.ShapeFieldNode{
				Shape: &shape,
			}
		} else {
			assertion := p.parseAssertionChain(true)
			log.Trace(fmt.Sprintf("Parsed assertion chain: %s", assertion))
			value = ast.ShapeFieldNode{
				Assertion: &assertion,
			}
		}

		fields[name] = value

		// Handle commas between fields
		if !isParenthesis(p.current()) && p.current().Type != ast.TokenRBrace {
			p.expect(ast.TokenComma)
		}
	}

	p.expect(ast.TokenRBrace)

	return ast.ShapeNode{Fields: fields}
}
