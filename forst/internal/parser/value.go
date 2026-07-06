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
			ident := p.parseIdentifier()
			refName := string(ident.ID)
			if p.current().Type == ast.TokenLBrace && isShapeLiteralTypePrefix(refName) {
				typeIdent := ast.TypeIdent(refName)
				shape := p.parseShapeLiteral(ShapeLiteralOpts{BaseType: &typeIdent})
				return ast.ReferenceNode{Value: shape}
			}
			return ast.ReferenceNode{
				Value: ast.VariableNode{
					Ident: ident,
				},
			}
		case ast.TokenLBrace:
			// Handle struct literal reference
			// Check for identifier before left brace (e.g., MyShape { ... })
			baseTypeIdent := p.parseTypeIdent()
			shapeLiteral := p.parseShapeLiteral(ShapeLiteralOpts{BaseType: baseTypeIdent})
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
			shape := p.parseShapeLiteral(ShapeLiteralOpts{BaseType: &typeIdent})
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
		shape := p.parseShapeLiteral(ShapeLiteralOpts{})
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

// parseSelectorName parses a single segment after `.` in a selector chain. Go method names
// such as String collide with Forst type keywords; allow those tokens as member names here.
func (p *Parser) parseSelectorName() ast.Token {
	tok := p.current()
	switch tok.Type {
	case ast.TokenIdentifier:
		p.advance()
		return tok
	case ast.TokenString:
		p.advance()
		return ast.Token{Type: ast.TokenIdentifier, Value: "String", Line: tok.Line, Column: tok.Column}
	case ast.TokenInt:
		p.advance()
		return ast.Token{Type: ast.TokenIdentifier, Value: "Int", Line: tok.Line, Column: tok.Column}
	case ast.TokenFloat:
		p.advance()
		return ast.Token{Type: ast.TokenIdentifier, Value: "Float", Line: tok.Line, Column: tok.Column}
	case ast.TokenBool:
		p.advance()
		return ast.Token{Type: ast.TokenIdentifier, Value: "Bool", Line: tok.Line, Column: tok.Column}
	default:
		return p.expect(ast.TokenIdentifier)
	}
}

func (p *Parser) parseIdentifier() ast.Ident {
	first := p.current()
	identifier := ast.Identifier(first.Value)
	p.advance()

	last := first
	for p.current().Type == ast.TokenDot {
		p.advance() // Consume dot
		nextIdent := p.parseSelectorName()
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
