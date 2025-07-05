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
		shape := p.parseShape(nil)
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
	p.expect(ast.TokenColon)

	paramType := p.parseParameterType()

	return ast.DestructuredParamNode{
		Fields: fields,
		Type:   paramType,
	}
}

func (p *Parser) parseSimpleParameter() ast.ParamNode {
	name := p.expect(ast.TokenIdentifier).Value
	// Remove colon requirement - use Go-style parameter declarations

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
		shape := p.parseShape(nil)
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
		// Support both single and parenthesized multiple return types
		if p.current().Type == ast.TokenLParen {
			p.advance() // Consume '('
			for {
				typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
				returnType = append(returnType, typ)
				if p.current().Type == ast.TokenComma {
					p.advance()
				} else {
					break
				}
			}
			p.expect(ast.TokenRParen)
		} else {
			returnType = append(returnType, p.parseType(TypeIdentOpts{AllowLowercaseTypes: false}))
		}
	}
	return returnType
}

func (p *Parser) parseReturnStatement() ast.ReturnNode {
	p.advance() // Move past `return`

	// Parse multiple return values
	values := []ast.ExpressionNode{}

	// Parse first expression
	if p.current().Type != ast.TokenSemicolon && p.current().Type != ast.TokenRBrace {
		values = append(values, p.parseExpression())
	}

	// Parse additional expressions separated by commas
	for p.current().Type == ast.TokenComma {
		p.advance() // Consume comma
		values = append(values, p.parseExpression())
	}

	return ast.ReturnNode{
		Values: values,
		Type:   ast.TypeNode{Ident: ast.TypeImplicit},
	}
}

func (p *Parser) parseFunctionBody() []ast.Node {
	return p.parseBlock()
}

// Parse a function definition
func (p *Parser) parseFunctionDefinition() ast.FunctionNode {
	p.expect(ast.TokenFunc)               // Expect `fn`
	name := p.expect(ast.TokenIdentifier) // Function name

	p.context.ScopeStack.CurrentScope().FunctionName = name.Value

	params := p.parseFunctionSignature() // Parse function parameters

	returnType := p.parseReturnType()

	body := p.parseFunctionBody()

	node := ast.FunctionNode{
		Ident:       ast.Ident{ID: ast.Identifier(name.Value)},
		ReturnTypes: returnType,
		Params:      params,
		Body:        body,
	}

	return node
}
