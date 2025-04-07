package parser

import (
	"fmt"

	"forst/pkg/ast"
)

// Mutable context for the parser to track the current state
type Context struct {
	Package  *ast.PackageNode
	Function *ast.FunctionNode
	Scope    *Scope
	FilePath string
}

type Parser struct {
	tokens       []ast.Token
	currentIndex int
	context      *Context
}

// Create a new parser instance
func NewParser(tokens []ast.Token, filePath string) *Parser {
	return &Parser{
		tokens:       tokens,
		currentIndex: 0,
		context:      &Context{FilePath: filePath},
	}
}

// Get the current token
func (p *Parser) current() ast.Token {
	if p.currentIndex < len(p.tokens) {
		return p.tokens[p.currentIndex]
	}
	return ast.Token{Type: ast.TokenEOF, Value: ""}
}

// Get the token n positions ahead (defaults to next token if n not specified)
func (p *Parser) peek(n ...int) ast.Token {
	offset := 1
	if len(n) > 0 {
		offset = n[0]
	}
	if p.currentIndex+offset < len(p.tokens) {
		return p.tokens[p.currentIndex+offset]
	}
	return ast.Token{Type: ast.TokenEOF, Value: ""}
}

// Advance to the next token
func (p *Parser) advance() ast.Token {
	p.currentIndex++
	return p.current()
}

// Expect a token and advance
func (p *Parser) expect(tokenType ast.TokenIdent) ast.Token {
	token := p.current()
	if token.Type != tokenType {
		panic(parseErrorWithValue(token, fmt.Sprintf("Expected token type '%s' but got '%s'", tokenType, token.Type)))
	}
	p.advance()
	return token
}
