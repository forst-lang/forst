package parser

import (
	"fmt"
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
			Name: name.Value,
			Type: paramType,
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
		Type:  returnExpression.ImplicitType(),
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

	explicitReturnType := p.parseReturnType()

	body := p.parseFunctionBody(context)

	implicitReturnType := ast.TypeNode{Name: ast.TypeVoid}
	for _, node := range body {
		if returnNode, ok := node.(ast.ReturnNode); ok {
			implicitReturnType = returnNode.Type
			break
		}
	}

	if !explicitReturnType.IsImplicit() && implicitReturnType.Name != explicitReturnType.Name {
		panic(fmt.Sprintf(
			"\nParse error in %s:%d:%d at line %d, column %d:\n"+
				"Function '%s' has return type mismatch: %s != %s",
			name.Path, name.Line, name.Column, name.Line, name.Column, name.Value,
			implicitReturnType.Name, explicitReturnType.Name,
		))
	}

	return ast.FunctionNode{
		Name:               name.Value,
		ExplicitReturnType: explicitReturnType,
		ImplicitReturnType: implicitReturnType,
		Params:             params,
		Body:               body,
	}
}
