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
		if nextToken.Type == ast.TokenIdentifier {
			ref := p.expect(ast.TokenIdentifier)
			return ast.ReferenceNode{
				Value: ast.VariableNode{
					Ident: ast.Ident{ID: ast.Identifier(ref.Value)},
				},
			}
		} else if nextToken.Type == ast.TokenLBrace {
			// Handle struct literal reference
			// Check for identifier before left brace (e.g., MyShape { ... })
			baseTypeIdent := p.parseTypeIdent()
			shapeLiteral := p.parseShape(baseTypeIdent)
			return ast.ReferenceNode{
				Value: shapeLiteral,
			}
		}
		p.FailWithParseError(nextToken, "Expected identifier or shape literal after &")
	case ast.TokenIdentifier:
		identifier := p.parseIdentifier()

		if p.current().Type == ast.TokenLBrace {
			typeIdent := ast.TypeIdent(string(identifier))
			shape := p.parseShape(&typeIdent)
			return shape
		}

		return ast.VariableNode{
			Ident: ast.Ident{ID: identifier},
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
		shape := p.parseShape(nil)
		return shape
	case ast.TokenStar:
		p.advance() // Consume *
		identifier := p.parseIdentifier()
		return ast.DereferenceNode{
			Value: ast.VariableNode{
				Ident: ast.Ident{ID: identifier},
			},
		}
	default:
		return p.parseLiteral()
	}
	panic("Reached unreachable path")
}

func (p *Parser) parseIdentifier() ast.Identifier {
	identifier := ast.Identifier(p.current().Value)

	p.advance() // Consume identifier

	// Keep chaining identifiers with dots until we hit something else
	for p.current().Type == ast.TokenDot {
		p.advance() // Consume dot
		nextIdent := p.expect(ast.TokenIdentifier)
		identifier = ast.Identifier(string(identifier) + "." + nextIdent.Value)
	}

	return identifier
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
