package main

import "fmt"

// Parser struct to keep track of tokens
type Parser struct {
	tokens  []Token
	currentIndex int
}

// Create a new parser instance
func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens, currentIndex: 0}
}

// Get the current token
func (p *Parser) current() Token {
	if p.currentIndex < len(p.tokens) {
		return p.tokens[p.currentIndex]
	}
	return Token{TokenEOF, ""}
}

// Advance to the next token
func (p *Parser) advance() Token {
	p.currentIndex++
	return p.current()
}

// Expect a token and advance
func (p *Parser) expect(tokenType string) Token {
	token := p.current()
	if token.Type != tokenType {
		panic(fmt.Sprintf("Expected %s but got %s", tokenType, token.Type))
	}
	p.advance()
	return token
}

// Parse a function definition
func (p *Parser) parseFunc() FuncNode {
	p.expect(TokenFunc)            // Expect `fn`
	name := p.expect(TokenIdent)   // Function name
	p.expect(TokenArrow)           // Expect `->`
	returnType := p.expect(TokenIdent).Value // Return type

	p.expect(TokenLBrace) // Expect `{`
	body := []Node{}

	// Parse function body dynamically
	for p.current().Type != TokenRBrace && p.current().Type != TokenEOF {
		token := p.current()

		if token.Type == TokenAssert {
			p.advance() // Move past `assert`
			condition := p.expect(TokenIdent).Value
			p.expect(TokenOr) // Expect `or`
			errorType := p.expect(TokenIdent).Value
			body = append(body, AssertNode{Condition: condition, ErrorType: errorType})
		} else if token.Type == TokenReturn {
			p.advance() // Move past `return`
			value := p.expect(TokenString).Value
			body = append(body, ReturnNode{Value: value})
		} else {
			panic(fmt.Sprintf("Unexpected token: %s", token.Value))
		}
	}

	p.expect(TokenRBrace) // Expect `}`
	return FuncNode{Name: name.Value, ReturnType: returnType, Body: body}
}

// Parse the tokens into an AST
func (p *Parser) Parse() FuncNode {
	return p.parseFunc()
}