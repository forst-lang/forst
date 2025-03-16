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
		paramType := p.parseType()

		params = append(params, ast.ParamNode{
			Ident: ast.Ident{Id: ast.Identifier(name.Value)},
			Type:  paramType,
		})

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

func (p *Parser) parseReturnType() ast.TypeNode {
	returnType := ast.TypeNode{Name: ast.TypeImplicit}
	if p.current().Type == ast.TokenColon {
		p.advance() // Consume the colon
		returnType = p.parseType()
	}
	return returnType
}

func (p *Parser) parseReturnStatement(context *Context) ast.ReturnNode {
	p.advance() // Move past `return`

	returnExpression := p.parseExpression(context)

	return ast.ReturnNode{
		Value: returnExpression,
		Type:  ast.TypeNode{Name: ast.TypeImplicit},
	}
}

func (p *Parser) parseFunctionBody(context *Context) []ast.Node {
	return p.parseBlock(&BlockContext{AllowReturn: true}, context)
}

// Parse a function definition
func (p *Parser) parseFunctionDefinition(context *Context) ast.FunctionNode {
	p.expect(ast.TokenFunction)           // Expect `fn`
	name := p.expect(ast.TokenIdentifier) // Function name

	context.Scope.FunctionName = &name.Value

	params := p.parseFunctionSignature() // Parse function parameters

	returnType := p.parseReturnType()

	body := p.parseFunctionBody(context)

	return ast.FunctionNode{
		Ident:      ast.Ident{Id: ast.Identifier(name.Value)},
		ReturnType: returnType,
		Params:     params,
		Body:       body,
	}
}
