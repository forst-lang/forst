package parser

import (
	"forst/internal/ast"
	"strconv"
	"strings"
)

// parseLiteral parses basic literal types (string, int, float, bool)
func (p *Parser) parseLiteral() ast.LiteralNode {
	neg := false
	if p.current().Type == ast.TokenMinus {
		neg = true
		p.advance()
	}

	token := p.current()
	p.advance() // Consume the token

	switch token.Type {
	case ast.TokenStringLiteral:
		if neg {
			p.FailWithParseError(token, "Invalid use of unary minus before string literal")
		}
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
			if neg {
				floatVal = -floatVal
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
		if neg {
			intVal = -intVal
		}
		return ast.IntLiteralNode{
			Value: intVal,
		}

	case ast.TokenFloatLiteral:
		s := strings.TrimSuffix(token.Value, "f")
		floatVal, err := strconv.ParseFloat(s, 64)
		if err != nil {
			p.FailWithParseError(token, "Invalid float literal")
		}
		if neg {
			floatVal = -floatVal
		}
		return ast.FloatLiteralNode{
			Value: floatVal,
		}

	case ast.TokenTrue:
		if neg {
			p.FailWithParseError(token, "Invalid use of unary minus before boolean literal")
		}
		return ast.BoolLiteralNode{
			Value: true,
		}

	case ast.TokenFalse:
		if neg {
			p.FailWithParseError(token, "Invalid use of unary minus before boolean literal")
		}
		return ast.BoolLiteralNode{
			Value: false,
		}

	case ast.TokenLBracket:
		if neg {
			p.FailWithParseError(token, "Invalid use of unary minus before array literal")
		}
		items := []ast.LiteralNode{}
		for p.current().Type != ast.TokenRBracket {
			items = append(items, p.parseLiteral())

			if p.current().Type == ast.TokenComma {
				p.advance() // Consume comma
			}
		}
		rbrack := p.expect(ast.TokenRBracket)

		var arrayType ast.TypeNode
		next := p.current()
		// Optional `[elem...]T` suffix: only when `T` is on the same line as `]`. Otherwise a
		// closing `]` at end of line would consume the next statement's identifier (e.g. `xs` in
		// `ys := [...]\n  xs := [...]`).
		if next.Type == ast.TokenIdentifier && next.Line == rbrack.Line {
			arrayType = ast.TypeNode{
				Ident: ast.TypeIdent(next.Value),
			}
			p.advance()
		} else {
			arrayType = ast.TypeNode{Ident: ast.TypeImplicit}
		}

		return ast.ArrayLiteralNode{
			Value: items,
			Type:  arrayType,
		}

	case ast.TokenNil:
		if neg {
			p.FailWithParseError(token, "Invalid use of unary minus before nil")
		}
		return ast.NilLiteralNode{}

	default:
		p.FailWithUnexpectedToken(token, "a literal")
	}
	panic("Reached unreachable path")
}
