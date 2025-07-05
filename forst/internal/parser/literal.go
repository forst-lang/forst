package parser

import (
	"forst/internal/ast"
	"strconv"
	"strings"
)

// parseLiteral parses basic literal types (string, int, float, bool)
func (p *Parser) parseLiteral() ast.LiteralNode {
	token := p.current()
	p.advance() // Consume the token

	switch token.Type {
	case ast.TokenStringLiteral:
		// Remove quotes from the string value
		value := token.Value
		if len(value) >= 2 {
			value = value[1 : len(value)-1] // Remove surrounding quotes
		}
		return ast.StringLiteralNode{
			Value: value,
		}

	case ast.TokenIntLiteral:
		value := token.Value

		// Check if it's a float (has a dot token after)
		if p.current().Type == ast.TokenDot {
			p.advance() // Consume dot

			floatSuffix := p.expect(ast.TokenFloatLiteral)
			decimalVal := value
			mantissa := strings.TrimSuffix(floatSuffix.Value, "f")
			floatVal, err := strconv.ParseFloat(decimalVal+"."+mantissa, 64)
			if err != nil {
				p.FailWithParseError(token, "Invalid float literal")
			}
			return ast.FloatLiteralNode{
				Value: floatVal,
			}
		}

		// It's an integer
		intVal, err := strconv.ParseInt(token.Value, 10, 64)
		if err != nil {
			p.FailWithParseError(token, "Invalid integer literal")
		}
		return ast.IntLiteralNode{
			Value: intVal,
		}

	case ast.TokenTrue:
		return ast.BoolLiteralNode{
			Value: true,
		}

	case ast.TokenFalse:
		return ast.BoolLiteralNode{
			Value: false,
		}

	case ast.TokenLBracket:
		items := []ast.LiteralNode{}
		for p.current().Type != ast.TokenRBracket {
			items = append(items, p.parseLiteral())

			if p.current().Type == ast.TokenComma {
				p.advance() // Consume comma
			}
		}
		p.expect(ast.TokenRBracket)

		var arrayType ast.TypeNode
		if p.current().Type == ast.TokenIdentifier {
			// Parse array type annotation
			arrayType = ast.TypeNode{
				Ident: ast.TypeIdent(p.current().Value),
			}
			p.advance() // Consume type identifier
		} else {
			arrayType = ast.TypeNode{Ident: ast.TypeImplicit}
		}

		return ast.ArrayLiteralNode{
			Value: items,
			Type:  arrayType,
		}

	case ast.TokenNil:
		p.advance()
		return ast.NilLiteralNode{}

	default:
		p.FailWithUnexpectedToken(token, "a literal")
	}
	panic("Reached unreachable path")
}
