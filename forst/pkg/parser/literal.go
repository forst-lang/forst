package parser

import (
	"forst/pkg/ast"
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
				panic(parseErrorWithValue(token, "Invalid float literal"))
			}
			return ast.FloatLiteralNode{
				Value: floatVal,
			}
		}

		// It's an integer
		intVal, err := strconv.ParseInt(token.Value, 10, 64)
		if err != nil {
			panic(parseErrorWithValue(token, "Invalid integer literal"))
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

	default:
		panic(unexpectedTokenMessage(token, "a literal"))
	}
}
