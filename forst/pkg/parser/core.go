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

// Get the next token
func (p *Parser) peek() ast.Token {
	if p.currentIndex+1 < len(p.tokens) {
		return p.tokens[p.currentIndex+1]
	}
	return ast.Token{Type: ast.TokenEOF, Value: ""}
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
		panic(parseErrorWithValue(token, fmt.Sprintf("Expected token type '%s' but got '%s'", tokenType, token.Type)))
	}
	p.advance()
	return token
}
