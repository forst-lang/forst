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

func (p *Parser) parseReturnStatement() ast.ReturnNode {
	p.advance() // Move past `return`

	returnExpression := p.parseExpression()

	return ast.ReturnNode{
		Value: returnExpression,
		Type:  returnExpression.ImplicitType(),
	}
}

func (p *Parser) parseEnsureStatement(context *Context) ast.EnsureNode {
	p.advance() // Move past `ensure`

	variable := p.expect(ast.TokenIdentifier).Value

	p.expect(ast.TokenIs)

	condition := p.parseAssertionChain(context)

	if !context.IsMainFunction() || p.current().Type == ast.TokenOr {
		p.expect(ast.TokenOr) // Expect `or`
		errorType := p.expect(ast.TokenIdentifier).Value
		// Parse error arguments
		p.expect(ast.TokenLParen)
		var args []ast.ExpressionNode
		for p.current().Type != ast.TokenRParen {
			args = append(args, p.parseExpression())
			if p.current().Type == ast.TokenComma {
				p.advance()
			}
		}
		p.expect(ast.TokenRParen)
		return ast.EnsureNode{Variable: variable, Assertion: condition, ErrorType: &errorType, ErrorArgs: args}
	}

	errorType := p.expect(ast.TokenIdentifier).Value
	return ast.EnsureNode{Variable: variable, Assertion: condition, ErrorType: &errorType}
}

func (p *Parser) parseFunctionBody(context *Context) []ast.Node {
	body := []ast.Node{}

	p.expect(ast.TokenLBrace) // Expect `{`

	// Parse function body dynamically
	for p.current().Type != ast.TokenRBrace && p.current().Type != ast.TokenEOF {
		token := p.current()

		if token.Type == ast.TokenEnsure {
			body = append(body, p.parseEnsureStatement(context))
		} else if token.Type == ast.TokenReturn {
			body = append(body, p.parseReturnStatement())
		} else {
			token := p.current()
			panic(fmt.Sprintf(
				"\nParse error in %s:%d:%d at line %d, column %d:\n"+
					"Unexpected token in function body: '%s'\n"+
					"Token value: '%s'",
				token.Path,
				token.Line,
				token.Column,
				token.Line,
				token.Column,
				token.Type,
				token.Value,
			))
		}
	}

	p.expect(ast.TokenRBrace) // Expect `}`

	return body
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
