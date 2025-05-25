package parser

import (
	"forst/internal/ast"

	log "github.com/sirupsen/logrus"
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
	return p.parseType()
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
	name := p.expect(ast.TokenIdentifier)
	p.expect(ast.TokenColon)

	paramType := p.parseParameterType()
	log.Trace("parsed param type", paramType)

	param := ast.SimpleParamNode{
		Ident: ast.Ident{Id: ast.Identifier(name.Value)},
		Type:  paramType,
	}

	return param
}

func (p *Parser) parseParameter() ast.ParamNode {
	switch p.current().Type {
	case ast.TokenIdentifier:
		return p.parseSimpleParameter()
	case ast.TokenLBrace:
		return p.parseDestructuredParameter()
	default:
		panic(parseErrorMessage(p.current(), "Expected parameter"))
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
			logParsedNodeWithMessage(param, "Parsed destructured function param")
		default:
			logParsedNodeWithMessage(param, "Parsed function param")
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
		returnType = append(returnType, p.parseType())
	}
	return returnType
}

func (p *Parser) parseReturnStatement() ast.ReturnNode {
	p.advance() // Move past `return`

	returnExpression := p.parseExpression()

	return ast.ReturnNode{
		Value: returnExpression,
		Type:  ast.TypeNode{Ident: ast.TypeImplicit},
	}
}

func (p *Parser) parseFunctionBody() []ast.Node {
	return p.parseBlock(&BlockContext{AllowReturn: true})
}

// Parse a function definition
func (p *Parser) parseFunctionDefinition() ast.FunctionNode {
	p.expect(ast.TokenFunction)           // Expect `fn`
	name := p.expect(ast.TokenIdentifier) // Function name

	p.context.Scope.functionName = name.Value

	params := p.parseFunctionSignature() // Parse function parameters

	returnType := p.parseReturnType()

	body := p.parseFunctionBody()

	node := ast.FunctionNode{
		Ident:       ast.Ident{Id: ast.Identifier(name.Value)},
		ReturnTypes: returnType,
		Params:      params,
		Body:        body,
	}

	return node
}
