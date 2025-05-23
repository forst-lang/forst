package parser

import (
	"forst/pkg/ast"
)

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
