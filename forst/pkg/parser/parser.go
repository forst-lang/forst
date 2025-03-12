package parser

import (
	"fmt"

	"forst/pkg/ast"
)

// Parser struct to keep track of tokens
type Parser struct {
	tokens  []ast.Token
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

// Advance to the next token
func (p *Parser) advance() ast.Token {
	p.currentIndex++
	return p.current()
}

// Expect a token and advance
func (p *Parser) expect(tokenType string) ast.Token {
	token := p.current()
	if token.Type != tokenType {
		panic(fmt.Sprintf("Expected %s but got %s", tokenType, token.Type))
	}
	p.advance()
	return token
}

// Parse a function definition
func (p *Parser) parseFunc() ast.FuncNode {
	p.expect(ast.TokenFunc)            // Expect `fn`
	name := p.expect(ast.TokenIdent)   // Function name
	p.expect(ast.TokenArrow)           // Expect `->`
	returnType := p.expect(ast.TokenIdent).Value // Return type

	p.expect(ast.TokenLBrace) // Expect `{`
	body := []ast.Node{}

	// Parse function body dynamically
	for p.current().Type != ast.TokenRBrace && p.current().Type != ast.TokenEOF {
		token := p.current()

		if token.Type == ast.TokenAssert {
			p.advance() // Move past `assert`
			condition := p.expect(ast.TokenIdent).Value
			p.expect(ast.TokenOr) // Expect `or`
			errorType := p.expect(ast.TokenIdent).Value
			body = append(body, ast.AssertNode{Condition: condition, ErrorType: errorType})
		} else if token.Type == ast.TokenReturn {
			p.advance() // Move past `return`
			value := p.expect(ast.TokenString).Value
			body = append(body, ast.ReturnNode{Value: value})
		} else {
			panic(fmt.Sprintf("Unexpected token: %s", token.Value))
		}
	}

	p.expect(ast.TokenRBrace) // Expect `}`
	return ast.FuncNode{Name: name.Value, ReturnType: returnType, Body: body}
}

// Parse the tokens into an AST
func (p *Parser) Parse() ast.FuncNode {
	return p.parseFunc()
}