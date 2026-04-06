package parser

import (
	"forst/internal/ast"
)

func (p *Parser) parseValue() ast.ValueNode {
	token := p.current()

	switch token.Type {
	case ast.TokenBitwiseAnd:
		p.advance() // Consume &
		nextToken := p.current()
		switch nextToken.Type {
		case ast.TokenIdentifier:
			ref := p.expect(ast.TokenIdentifier)
			return ast.ReferenceNode{
				Value: ast.VariableNode{
					Ident: ast.Ident{ID: ast.Identifier(ref.Value), Span: ast.SpanFromToken(ref)},
				},
			}
		case ast.TokenLBrace:
			// Handle struct literal reference
			// Check for identifier before left brace (e.g., MyShape { ... })
			baseTypeIdent := p.parseTypeIdent()
			shapeLiteral := p.parseShapeLiteral(baseTypeIdent, false)
			return ast.ReferenceNode{
				Value: shapeLiteral,
			}
		default:
			p.FailWithParseError(nextToken, "Expected identifier or shape literal after &")
		}
	case ast.TokenIdentifier:
		ident := p.parseIdentifier()

		if p.current().Type == ast.TokenLBrace && isShapeLiteralTypePrefix(string(ident.ID)) {
			typeIdent := ast.TypeIdent(string(ident.ID))
			shape := p.parseShapeLiteral(&typeIdent, false)
			return shape
		}

		return ast.VariableNode{
			Ident: ident,
		}
	case ast.TokenMap:
		p.advance() // Consume map keyword

		// Parse key type
		p.expect(ast.TokenLBracket) // Expect [
		keyType := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
		p.expect(ast.TokenRBracket) // Expect ]

		// Parse value type
		valueType := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})

		// Parse map entries
		p.expect(ast.TokenLBrace) // Expect {
		entries := []ast.MapEntryNode{}

		for p.current().Type != ast.TokenRBrace {
			key := p.parseValue()
			p.expect(ast.TokenColon)
			value := p.parseValue()

			entries = append(entries, ast.MapEntryNode{
				Key:   key,
				Value: value,
			})

			if p.current().Type == ast.TokenComma {
				p.advance() // Consume comma
			}
		}
		p.expect(ast.TokenRBrace) // Expect }

		return ast.MapLiteralNode{
			Entries: entries,
			Type: ast.TypeNode{
				Ident:      ast.TypeMap,
				TypeParams: []ast.TypeNode{keyType, valueType},
			},
		}
	case ast.TokenLBrace:
		// Handle shape literal
		shape := p.parseShapeLiteral(nil, false)
		return shape
	case ast.TokenStar:
		p.advance() // Consume *
		ident := p.parseIdentifier()
		return ast.DereferenceNode{
			Value: ast.VariableNode{
				Ident: ident,
			},
		}
	default:
		return p.parseLiteral()
	}
	panic("Reached unreachable path")
}

func (p *Parser) parseIdentifier() ast.Ident {
	first := p.current()
	identifier := ast.Identifier(first.Value)
	p.advance()

	last := first
	for p.current().Type == ast.TokenDot {
		p.advance() // Consume dot
		nextIdent := p.expect(ast.TokenIdentifier)
		identifier = ast.Identifier(string(identifier) + "." + nextIdent.Value)
		last = nextIdent
	}

	return ast.Ident{
		ID:   identifier,
		Span: ast.SpanBetweenTokens(first, last),
	}
}

// parseTypeIdent parses a type identifier
func (p *Parser) parseTypeIdent() *ast.TypeIdent {
	if p.current().Type != ast.TokenIdentifier {
		p.FailWithParseError(p.current(), "expected identifier")
	}
	name := p.current().Value
	p.advance()
	typeIdent := ast.TypeIdent(name)
	return &typeIdent
}
