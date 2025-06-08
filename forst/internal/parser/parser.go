package parser

import (
	"fmt"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

// Context is a mutable context for the parser to track the current state
type Context struct {
	Package    *ast.PackageNode
	Function   *ast.FunctionNode
	ScopeStack *ScopeStack
	FilePath   string
}

// Parser represents the parser for the Forst language
type Parser struct {
	tokens       []ast.Token
	currentIndex int
	context      *Context
	log          *logrus.Logger
}

// New creates a new parser instance
func New(tokens []ast.Token, filePath string, log *logrus.Logger) *Parser {
	scopeStack := NewScopeStack(log)
	context := Context{
		FilePath:   filePath,
		ScopeStack: scopeStack,
	}
	if log == nil {
		log = logrus.New()
		log.Warnf("No logger provided, using default logger")
	}
	return &Parser{
		tokens:       tokens,
		currentIndex: 0,
		context:      &context,
		log:          log,
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
		p.FailWithParseError(token, fmt.Sprintf("Expected token type '%s' but got '%s'", tokenType, token.Type))
	}
	p.advance()
	return token
}

func (p *Parser) FailWithUnexpectedToken(token ast.Token, message string) {
	p.FailWithParseError(token, unexpectedTokenMessage(token, message))
}

func (p *Parser) FailWithParseError(token ast.Token, message string) {
	p.log.Fatalf("%s", parseErrorMessage(token, message))
}
