package parser

import (
	"forst/pkg/ast"
)

// Parse a type definition
func (p *Parser) parseType() ast.TypeNode {
	token := p.current()

	// TODO: Support slices and maps

	switch token.Type {
	case ast.TokenString:
		p.advance()
		return ast.TypeNode{Ident: ast.TypeString}
	case ast.TokenInt:
		p.advance()
		return ast.TypeNode{Ident: ast.TypeInt}
	case ast.TokenFloat:
		p.advance()
		return ast.TypeNode{Ident: ast.TypeFloat}
	case ast.TokenBool:
		p.advance()
		return ast.TypeNode{Ident: ast.TypeBool}
	case ast.TokenVoid:
		p.advance()
		return ast.TypeNode{Ident: ast.TypeVoid}
	default:
		// Parse first segment (could be package name or type)
		firstSegment := p.expectCustomTypeIdentifierOrPackageName().Value

		// Check if it's a package name
		if p.current().Type == ast.TokenDot && p.peek(2).Type != ast.TokenLParen {
			p.advance() // Consume dot
			typeName := p.expectCustomTypeIdentifier().Value
			qualifiedName := firstSegment + "." + typeName
			return ast.TypeNode{Ident: ast.TypeIdent(qualifiedName)}
		}

		return ast.TypeNode{Ident: ast.TypeIdent(firstSegment)}
	}
}

func isPossibleTypeIdentifier(token ast.Token) bool {
	if token.Type == ast.TokenIdentifier {
		return isCapitalCase(token.Value)
	}

	return token.Type == ast.TokenString ||
		token.Type == ast.TokenInt ||
		token.Type == ast.TokenFloat ||
		token.Type == ast.TokenBool ||
		token.Type == ast.TokenVoid
}
