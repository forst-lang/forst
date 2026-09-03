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

	case ast.TokenRuneLiteral:
		if neg {
			p.FailWithParseError(token, "Invalid use of unary minus before rune literal")
		}
		content := token.Value
		if len(content) >= 2 {
			content = content[1 : len(content)-1]
		}
		r, _, tail, err := strconv.UnquoteChar(content, '\'')
		if err != nil || tail != "" {
			p.FailWithParseError(token, "illegal rune literal")
		}
		return ast.RuneLiteralNode{
			Value: int64(r),
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

		// It's an integer (decimal, 0x hex, 0o octal, 0b binary — Go-compatible)
		intVal, err := strconv.ParseInt(token.Value, 0, 64)
		if err != nil {
			p.FailWithParseError(token, "Invalid integer literal")
		}
		if neg {
			intVal = -intVal
		}
		return ast.IntLiteralNode{
			Value: intVal,
			Raw:   token.Value,
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
		items := []ast.ExpressionNode{}
		for p.current().Type != ast.TokenRBracket {
			items = append(items, p.parseExpression())

			if p.current().Type == ast.TokenComma {
				p.advance() // Consume comma
			}
		}
		rbrack := p.expect(ast.TokenRBracket)

		arrayType := p.parseOptionalArrayElementTypeSuffix(rbrack)
		if p.current().Type == ast.TokenLBrace {
			return p.parseBraceCompositeArrayLiteral(arrayType)
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

// parseBraceCompositeArrayLiteral parses `{ elem, elem, … }` after a typed `[]T` prefix.
// Nested `{…}` elements are treated as Go-style nested composites (not Forst shapes),
// so `[][]String{{"a","b"}}` works for table-driven tests.
func (p *Parser) parseBraceCompositeArrayLiteral(elemType ast.TypeNode) ast.ArrayLiteralNode {
	p.expect(ast.TokenLBrace)
	var items []ast.ExpressionNode
	for p.current().Type != ast.TokenRBrace {
		if p.current().Type == ast.TokenLBrace {
			nestedElem := ast.TypeNode{Ident: ast.TypeImplicit}
			if elemType.Ident == ast.TypeArray && len(elemType.TypeParams) == 1 {
				nestedElem = elemType.TypeParams[0]
			}
			items = append(items, p.parseBraceCompositeArrayLiteral(nestedElem))
		} else {
			items = append(items, p.parseExpression())
		}
		if p.current().Type == ast.TokenComma {
			p.advance()
		}
	}
	p.expect(ast.TokenRBrace)
	return ast.ArrayLiteralNode{
		Value: items,
		Type:  elemType,
	}
}

// parseOptionalArrayElementTypeSuffix parses optional `]T` on the same line as `]`.
// T may be a nested slice/array type (e.g. `[][]string{{…}}`).
func (p *Parser) parseOptionalArrayElementTypeSuffix(rbrack ast.Token) ast.TypeNode {
	next := p.current()
	if next.Line != rbrack.Line {
		return ast.TypeNode{Ident: ast.TypeImplicit}
	}
	switch next.Type {
	case ast.TokenLBracket, ast.TokenStar, ast.TokenMap, ast.TokenChan, ast.TokenFunc, ast.TokenArray:
		return p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
	case ast.TokenIdentifier:
		p.advance()
		return ast.TypeNode{Ident: ast.TypeIdent(next.Value)}
	case ast.TokenString:
		p.advance()
		return ast.NewBuiltinType(ast.TypeString)
	case ast.TokenInt:
		p.advance()
		return ast.NewBuiltinType(ast.TypeInt)
	case ast.TokenBool:
		p.advance()
		return ast.NewBuiltinType(ast.TypeBool)
	default:
		return ast.TypeNode{Ident: ast.TypeImplicit}
	}
}
