package parser

import (
	"forst/internal/ast"
)

// TypeIdentOpts configures how type identifiers are validated
type TypeIdentOpts struct {
	// Whether to allow lowercase type identifiers for built-in types
	// This is used for backwards compatibility with Golang
	// in places where we are in a valid Go context, such that all valid
	// Go code can be parsed as valid Forst code.
	AllowLowercaseTypes bool
}

func (p *Parser) expectCustomTypeIdentifier(opts TypeIdentOpts) ast.Token {
	token := p.expect(ast.TokenIdentifier)
	if !isPossibleTypeIdentifier(token, opts) {
		p.FailWithParseError(token, "Expected type identifier to start with an uppercase letter")
	}
	return token
}

func (p *Parser) expectCustomTypeIdentifierOrPackageName(opts TypeIdentOpts) ast.Token {
	// First, expect an identifier
	token := p.expect(ast.TokenIdentifier)

	// Check if this is a package name followed by a dot
	if p.current().Type == ast.TokenDot && isPossibleTypeIdentifier(p.peek(), opts) && p.peek(2).Type != ast.TokenLParen {
		// This is a package name, consume the dot and get the type name
		p.advance() // consume dot
		typeToken := p.expect(ast.TokenIdentifier)
		// Return a token with the qualified name
		return ast.Token{
			Type:   ast.TokenIdentifier,
			Value:  token.Value + "." + typeToken.Value,
			Line:   token.Line,
			Column: token.Column,
		}
	}

	// Validate the identifier as a type
	if !isPossibleTypeIdentifier(token, opts) {
		p.FailWithParseError(token, "Expected type identifier to start with an uppercase letter")
	}

	return token
}

func (p *Parser) parseTypeDef() *ast.TypeDefNode {
	p.expect(ast.TokenType)

	name := p.expectCustomTypeIdentifier(TypeIdentOpts{AllowLowercaseTypes: false})
	typeIdent := ast.TypeIdent(name.Value)

	p.expect(ast.TokenEquals)

	expr := p.parseTypeDefExpr()

	return &ast.TypeDefNode{
		Ident: typeIdent,
		Expr:  expr,
	}
}

func (p *Parser) parseTypeDefExpr() ast.TypeDefExpr {
	if p.current().Type == ast.TokenLParen {
		p.advance()
		expr := p.parseTypeDefExpr()
		p.expect(ast.TokenRParen)
		return expr
	}

	if p.current().Type == ast.TokenLBrace {
		shape := p.parseShapeType()
		return ast.TypeDefShapeExpr{Shape: shape}
	}

	var left ast.TypeDefExpr
	if p.peek().Type == ast.TokenDot {
		assertion := p.parseAssertionChain(true)
		left = ast.TypeDefAssertionExpr{
			Assertion: &assertion,
		}
	} else {
		typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})

		return &ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &typ.Ident,
			},
		}
	}

	operator := p.current()
	if operator.Type != ast.TokenBitwiseAnd && operator.Type != ast.TokenBitwiseOr {
		return left
	}

	p.advance()

	right := p.parseTypeDefExpr()

	return ast.TypeDefBinaryExpr{
		Left:  left,
		Op:    operator.Type,
		Right: right,
	}
}
