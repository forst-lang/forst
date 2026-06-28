package parser

import (
	"forst/internal/ast"
)

func (p *Parser) parseParameterType() ast.TypeNode {
	next := p.peek()
	// TODO: Even if the next token is a dot this could be a type without any constraints
	if next.Type == ast.TokenDot || next.Type == ast.TokenLParen {
		assertion := p.parseAssertionChain(true)
		return ast.TypeNode{
			Ident:     ast.TypeAssertion,
			Assertion: &assertion,
		}
	}
	// Disallow Shape({...}) wrapper for shape types
	if p.current().Type == ast.TokenIdentifier && p.current().Value == "Shape" {
		ident := p.expect(ast.TokenIdentifier)
		if p.current().Type == ast.TokenLParen {
			p.FailWithParseError(p.current(), "Shape({...}) wrapper is not allowed. Use {...} directly for shape types.")
		}
		return ast.TypeNode{
			Ident: ast.TypeIdent(ident.Value),
		}
	}
	// Allow direct {...} for shape types
	if p.current().Type == ast.TokenLBrace {
		shape := p.parseShapeType()
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
	return p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
}

func (p *Parser) parseDestructuredParameter() ast.ParamNode {
	p.expect(ast.TokenLBrace)
	fields := []string{}

	// Parse fields until we hit closing brace
	for p.current().Type != ast.TokenRBrace {
		name := p.expect(ast.TokenIdentifier)
		fields = append(fields, name.Value)

		// Handle comma between fields
		if p.current().Type == ast.TokenComma {
			p.advance()
		}
	}

	p.expect(ast.TokenRBrace)
	if p.current().Type == ast.TokenColon {
		p.advance()
	}

	paramType := p.parseParameterType()

	return ast.DestructuredParamNode{
		Fields: fields,
		Type:   paramType,
	}
}

func (p *Parser) parseSimpleParameter() ast.ParamNode {
	name := p.expect(ast.TokenIdentifier).Value
	// Go-style parameter declarations (name Type). Optional ':' before the type is accepted for legacy sources.
	if p.current().Type == ast.TokenColon {
		p.advance()
	}

	tok := p.current()
	if tok.Type == ast.TokenIdentifier && (p.peek().Type == ast.TokenDot || p.peek().Type == ast.TokenLParen) {
		assertion := p.parseAssertionChain(true)
		return ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier(name)},
			Type: ast.TypeNode{
				Ident:     ast.TypeAssertion,
				Assertion: &assertion,
			},
		}
	}

	if tok.Type == ast.TokenIdentifier && tok.Value == "Shape" {
		// Check if this is Shape({...})
		if p.peek().Type == ast.TokenLParen {
			p.FailWithParseError(tok, "Direct usage of Shape({...}) is not allowed. Use a shape type directly, e.g. { field: Type }.")
		}
		// Allow direct usage of Shape as a type name
		p.advance()
		typeIdent := ast.TypeIdent("Shape")
		return ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier(name)},
			Type:  ast.TypeNode{Ident: typeIdent},
		}
	}
	if tok.Type == ast.TokenLBrace {
		shape := p.parseShapeType()
		baseType := ast.TypeIdent(ast.TypeShape)
		return ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier(name)},
			Type: ast.TypeNode{
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
			},
		}
	}
	// Parse the type, which may include dots (e.g. AppMutation.Input)
	typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
	p.logParsedNodeWithMessage(typ, "Parsed parameter type, next token: "+p.current().Type.String()+" ("+p.current().Value+")")
	return ast.SimpleParamNode{
		Ident: ast.Ident{ID: ast.Identifier(name)},
		Type:  typ,
	}
}

func (p *Parser) parseParameter() ast.ParamNode {
	switch p.current().Type {
	case ast.TokenIdentifier:
		return p.parseSimpleParameter()
	case ast.TokenLBrace:
		return p.parseDestructuredParameter()
	default:
		p.FailWithParseError(p.current(), "Expected parameter")
		panic("Reached unreachable path")
	}
}

// Parse function parameters
func (p *Parser) parseFunctionSignature() []ast.ParamNode {
	p.expect(ast.TokenLParen)
	params := []ast.ParamNode{}

	// Handle empty parameter list
	if p.current().Type == ast.TokenRParen {
		p.advance()
		return params
	}

	// Parse parameters
	for {
		param := p.parseParameter()
		switch param.(type) {
		case ast.DestructuredParamNode:
			p.logParsedNodeWithMessage(param, "Parsed destructured function param")
		default:
			p.logParsedNodeWithMessage(param, "Parsed function param")
		}
		params = append(params, param)

		// Check if there are more parameters
		if p.current().Type == ast.TokenComma {
			p.advance()
		} else {
			break
		}
	}

	p.expect(ast.TokenRParen)
	return params
}

func (p *Parser) parseReturnType() []ast.TypeNode {
	returnType := []ast.TypeNode{}
	if p.current().Type == ast.TokenColon {
		p.advance() // Consume the colon
		// Single return type only (use Result(T, Error) or Tuple(...) instead of Go-style (T, U))
		if p.current().Type == ast.TokenLParen {
			p.FailWithParseError(p.current(), "multi-value return types are not supported; use Result(Success, Error) or Tuple(T1, ...)")
		}
		returnType = append(returnType, p.parseReturnTypeSingle())
	}
	return returnType
}

func (p *Parser) parseReturnTypeSingle() ast.TypeNode {
	// Special handling for shape types in return positions
	if p.current().Type == ast.TokenLBrace {
		shape := p.parseShapeType()
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
	return p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
}

func (p *Parser) parseReturnStatement() ast.ReturnNode {
	p.advance() // Move past `return`

	var values []ast.ExpressionNode
	if p.current().Type != ast.TokenSemicolon && p.current().Type != ast.TokenRBrace {
		values = append(values, p.parseExpression())
		if p.current().Type == ast.TokenComma {
			p.FailWithParseError(p.current(), "multiple return values are not supported; use a single Result(S, F) success value or delegate to a Result-returning call")
		}
	}

	return ast.ReturnNode{
		Values: values,
		Type:   ast.TypeNode{Ident: ast.TypeImplicit},
	}
}

func (p *Parser) parseFunctionBody() []ast.Node {
	return p.parseBlock()
}

// parseFunctionName reads a function or method name. `error` is allowed when it starts a signature.
// `use` is allowed as a function name when followed by `(` (the `use` statement keyword uses a different lookahead).
func (p *Parser) parseFunctionName() ast.Token {
	tok := p.current()
	switch tok.Type {
	case ast.TokenIdentifier:
		p.advance()
		return tok
	case ast.TokenError:
		if p.peek().Type == ast.TokenLParen {
			p.advance()
			return tok
		}
		p.FailWithParseError(tok, "expected function name")
	case ast.TokenUse:
		if p.peek().Type == ast.TokenLParen {
			p.advance()
			return tok
		}
		p.FailWithParseError(tok, "expected function name")
	default:
		p.FailWithParseError(tok, "expected function name")
	}
	panic("unreachable")
}

// Parse a function definition
func (p *Parser) parseFunctionDefinition() ast.FunctionNode {
	p.expect(ast.TokenFunc) // Expect `func`

	var receiver *ast.SimpleParamNode
	if p.current().Type == ast.TokenLParen {
		receiver = p.parseReceiver()
	}

	name := p.parseFunctionName() // Function name

	p.context.ScopeStack.CurrentScope().FunctionName = name.Value

	params := p.parseFunctionSignature() // Parse function parameters

	returnType := p.parseReturnType()

	body := p.parseFunctionBody()

	node := ast.FunctionNode{
		Receiver:    receiver,
		Ident:       ast.Ident{ID: ast.Identifier(name.Value), Span: ast.SpanFromToken(name)},
		ReturnTypes: returnType,
		Params:      params,
		Body:        body,
	}

	return node
}

func (p *Parser) parseReceiver() *ast.SimpleParamNode {
	p.expect(ast.TokenLParen)

	firstTok := p.expect(ast.TokenIdentifier)
	var recvName ast.Identifier
	var recvType ast.TypeNode

	if p.current().Type == ast.TokenIdentifier || p.current().Type == ast.TokenStar ||
		p.current().Type == ast.TokenMap || p.current().Type == ast.TokenLBracket {
		// func (l StdLogger) or func (l *T)
		recvName = ast.Identifier(firstTok.Value)
		recvType = p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
	} else {
		// func (StdLogger) or func (NopLogger) — type only, no receiver name
		p.currentIndex--
		recvType = p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
	}

	p.expect(ast.TokenRParen)

	var ident ast.Ident
	if recvName != "" {
		ident = ast.Ident{ID: recvName, Span: ast.SpanFromToken(firstTok)}
	}

	return &ast.SimpleParamNode{
		Ident: ident,
		Type:  recvType,
	}
}
