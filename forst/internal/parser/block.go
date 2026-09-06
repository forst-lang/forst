package parser

import (
	"forst/internal/ast"
)

func (p *Parser) parseBlock() []ast.Node {
	body, _ := p.parseBlockWithClose()
	return body
}

func (p *Parser) parseBlockWithClose() ([]ast.Node, ast.Token) {
	body := []ast.Node{}

	p.expect(ast.TokenLBrace) // Expect `{`

	for p.current().Type != ast.TokenRBrace && p.current().Type != ast.TokenEOF {
		for p.current().Type == ast.TokenComment {
			body = append(body, ast.CommentNode{Text: p.current().Value})
			p.advance()
		}
		if p.current().Type == ast.TokenRBrace || p.current().Type == ast.TokenEOF {
			break
		}
		body = append(body, p.parseBlockStatement()...)
	}

	rbrace := p.expect(ast.TokenRBrace) // Expect `}`
	return body, rbrace
}
