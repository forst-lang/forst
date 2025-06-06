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
			structLit := p.parseStructLiteral()
			return ast.ReferenceNode{
				Value: structLit,
			}
		}
		panic(parseErrorWithValue(nextToken, "Expected identifier or struct literal after &"))
	case ast.TokenIdentifier:
		p.advance() // Consume identifier
		identifier := ast.Identifier(token.Value)

		// Keep chaining identifiers with dots until we hit something else
		for p.current().Type == ast.TokenDot {
			p.advance() // Consume dot
			nextIdent := p.expect(ast.TokenIdentifier)
			identifier = ast.Identifier(string(identifier) + "." + nextIdent.Value)
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
		shape := p.parseShape()
		return shape
	default:
		return p.parseLiteral()
	}
}

// parseStructLiteral parses a struct literal value
func (p *Parser) parseStructLiteral() ast.ShapeNode {
	return p.parseShape()
}
