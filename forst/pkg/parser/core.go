package parser

import (
	"fmt"

	"forst/pkg/ast"
)

type Parser struct {
	tokens       []ast.Token
	currentIndex int
}

// Create a new parser instance
func NewParser(tokens []ast.Token) *Parser {
	return &Parser{tokens: tokens, currentIndex: 0}
}

// Get the current token
func (p *Parser) current() ast.Token {
	if p.currentIndex < len(p.tokens) {
		return p.tokens[p.currentIndex]
	}
	return ast.Token{Type: ast.TokenEOF, Value: ""}
}

func (p *Parser) previous() *ast.Token {
	if p.currentIndex > 0 {
		return &p.tokens[p.currentIndex-1]
	}
	return nil
}

// Advance to the next token
func (p *Parser) advance() ast.Token {
	p.currentIndex++
	return p.current()
}

// Expect a token and advance
func (p *Parser) expect(tokenType ast.TokenType) ast.Token {
	token := p.current()
	if token.Type != tokenType {
		panic(fmt.Sprintf(`
Parse error in %s:%d:%d (line %d, column %d):
Expected token type '%s' but got '%s'
Token value: '%s'`,
			token.Path,
			token.Line,
			token.Column,
			token.Line,
			token.Column,
			tokenType,
			token.Type,
			token.Value,
		))
	}
	p.advance()
	return token
}
