package parser

import (
	"forst/internal/ast"
)

// BlockContext represents the context for a block of statements
type BlockContext struct {
	AllowReturn bool
}

func (p *Parser) parseBlock(blockContext *BlockContext) []ast.Node {
	body := []ast.Node{}

	p.expect(ast.TokenLBrace) // Expect `{`

	// Parse function body dynamically
	for p.current().Type != ast.TokenRBrace && p.current().Type != ast.TokenEOF {
		body = append(body, p.parseBlockStatement(blockContext)...)
	}

	p.expect(ast.TokenRBrace) // Expect `}`

	return body
}
