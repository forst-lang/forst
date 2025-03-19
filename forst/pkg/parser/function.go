package parser

import (
	"forst/pkg/ast"
)

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
		name := p.expect(ast.TokenIdentifier)
		p.expect(ast.TokenColon)

		var paramType ast.TypeNode
		if p.peek().Type == ast.TokenDot || p.peek().Type == ast.TokenLParen {
			assertion := p.parseAssertionChain(false)
			paramType = ast.TypeNode{Name: ast.TypeAssertion, Assertion: &assertion}
		} else {
			paramType = p.parseType()
		}

		param := ast.ParamNode{
			Ident: ast.Ident{Id: ast.Identifier(name.Value)},
			Type:  paramType,
		}
		logParsedNodeWithMessage(param, "Parsed function param")
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
		Type:  ast.TypeNode{Name: ast.TypeImplicit},
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

	logParsedNode(node)

	return node
}
