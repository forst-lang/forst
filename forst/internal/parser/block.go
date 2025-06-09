package parser

import (
	"forst/internal/ast"
)

func (p *Parser) skipComments() {
	for p.current().Type == ast.TokenComment {
		p.advance()
	}
}

func (p *Parser) parseBlock() []ast.Node {
	body := []ast.Node{}

	p.expect(ast.TokenLBrace) // Expect `{`

	p.skipComments() // Skip comments at the start of the block

	// Parse function body dynamically
	for p.current().Type != ast.TokenRBrace && p.current().Type != ast.TokenEOF {
		body = append(body, p.parseBlockStatement()...)
		p.skipComments() // Skip comments between statements
	}

	p.expect(ast.TokenRBrace) // Expect `}`

	return body
}
