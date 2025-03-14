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
		// Check if it's a float (contains a decimal point)
		if strings.Contains(token.Value, ".") {
			// Check if it's a float32 (ends with 'f')
			isFloat32 := false
			value := token.Value
			if strings.HasSuffix(value, "f") {
				isFloat32 = true
				value = value[:len(value)-1] // Remove the 'f' suffix
			}

			floatVal, err := strconv.ParseFloat(value, 64)
			if err != nil {
				panic("Invalid float literal: " + token.Value)
			}

			if isFloat32 {
				panic("Float32 literals are not supported")
			}

			return ast.FloatLiteralNode{
				Value: floatVal,
			}
		}

		// It's an integer
		intVal, err := strconv.ParseInt(token.Value, 10, 64)
		if err != nil {
			panic("Invalid integer literal: " + token.Value)
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
		panic("Expected literal but got: " + token.Value)
	}
}
