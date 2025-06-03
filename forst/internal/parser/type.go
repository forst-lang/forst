package parser

import (
	"forst/internal/ast"
)

// Parse a type definition
func (p *Parser) parseType(opts TypeIdentOpts) ast.TypeNode {
	token := p.current()

	// Handle pointer types
	if token.Type == ast.TokenStar {
		p.advance() // consume *
		baseType := p.parseType(opts)
		return ast.TypeNode{
			Ident:      ast.TypePointer,
			TypeParams: []ast.TypeNode{baseType},
		}
	}

	// Handle shape types
	if token.Type == ast.TokenLBrace {
		shape := p.parseShape()
		baseType := ast.TypeIdent(ast.TypeShape)
		return ast.TypeNode{
			Ident: ast.TypeShape,
			Assertion: &ast.AssertionNode{
				BaseType: &baseType,
				Constraints: []ast.ConstraintNode{{
					Name: "Match",
					Args: []ast.ConstraintArgumentNode{{
						Shape: &shape,
					}},
				}},
			},
		}
	}

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
	case ast.TokenLBracket:
		// Parse slice type
		p.advance()                 // consume [
		p.expect(ast.TokenRBracket) // expect ]
		elementType := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
		return ast.TypeNode{
			Ident:      ast.TypeArray,
			TypeParams: []ast.TypeNode{elementType},
		}
	case ast.TokenMap:
		// Parse map type
		p.advance()                 // consume map
		p.expect(ast.TokenLBracket) // expect [
		keyType := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
		p.expect(ast.TokenRBracket) // expect ]
		valueType := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
		return ast.TypeNode{
			Ident:      ast.TypeMap,
			TypeParams: []ast.TypeNode{keyType, valueType},
		}
	case ast.TokenInterface:
		// Parse interface type
		p.advance() // consume interface keyword

		// Check if it's an empty interface
		if p.current().Type == ast.TokenLBrace {
			p.advance()               // consume {
			p.expect(ast.TokenRBrace) // expect }
			return ast.TypeNode{Ident: ast.TypeObject}
		}

		// Parse interface fields until closing brace
		p.expect(ast.TokenLBrace) // require opening brace
		for p.current().Type != ast.TokenRBrace {
			// Expect field name
			_ = p.expect(ast.TokenIdentifier)

			// Check if it's a method (has parentheses) or a field
			if p.current().Type == ast.TokenLParen {
				// Parse method parameters
				p.expect(ast.TokenLParen)

				// Parse parameters until closing parenthesis
				for p.current().Type != ast.TokenRParen {
					// Parse parameter name
					_ = p.expect(ast.TokenIdentifier)

					// Expect type annotation
					p.expect(ast.TokenColon)
					_ = p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})

					// Handle comma between parameters
					if p.current().Type == ast.TokenComma {
						p.advance()
					}
				}
				p.expect(ast.TokenRParen)

				// Expect return type
				p.expect(ast.TokenColon)
				_ = p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
			} else {
				// Regular field - expect type annotation
				p.expect(ast.TokenColon)
				_ = p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
			}
		}

		p.expect(ast.TokenRBrace) // require closing brace
		return ast.TypeNode{Ident: ast.TypeObject}
	default:
		// Parse first segment (could be package name or type)
		firstSegment := p.expectCustomTypeIdentifierOrPackageName(opts).Value

		// Check if it's a package name
		if p.current().Type == ast.TokenDot && p.peek(2).Type != ast.TokenLParen {
			p.advance() // Consume dot
			typeName := p.expectCustomTypeIdentifier(opts).Value
			qualifiedName := firstSegment + "." + typeName
			// If the next token is LParen, parse as type application/assertion
			if p.current().Type == ast.TokenLParen {
				assertion := p.parseAssertionChain(true)
				return ast.TypeNode{
					Ident:     ast.TypeAssertion,
					Assertion: &assertion,
				}
			}
			return ast.TypeNode{Ident: ast.TypeIdent(qualifiedName)}
		}

		// If the next token is LParen, parse as type application/assertion
		if p.current().Type == ast.TokenLParen {
			assertion := p.parseAssertionChain(true)
			return ast.TypeNode{
				Ident:     ast.TypeAssertion,
				Assertion: &assertion,
			}
		}

		return ast.TypeNode{Ident: ast.TypeIdent(firstSegment)}
	}
}

func isPossibleTypeIdentifier(token ast.Token, opts TypeIdentOpts) bool {
	if token.Type == ast.TokenIdentifier {
		return opts.AllowLowercaseTypes || isCapitalCase(token.Value)
	}

	return token.Type == ast.TokenString ||
		token.Type == ast.TokenInt ||
		token.Type == ast.TokenFloat ||
		token.Type == ast.TokenBool ||
		token.Type == ast.TokenVoid ||
		token.Type == ast.TokenLBracket ||
		token.Type == ast.TokenMap
}
