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

func (p *Parser) parseFunctionBody(explicitReturnType ast.TypeNode) []ast.Node {
	body := []ast.Node{}

	p.expect(ast.TokenLBrace) // Expect `{`

	// Parse function body dynamically
	for p.current().Type != ast.TokenRBrace && p.current().Type != ast.TokenEOF {
		token := p.current()

		if token.Type == ast.TokenEnsure {
			p.advance() // Move past `ensure`
			condition := p.expect(ast.TokenIdentifier).Value
			p.expect(ast.TokenOr) // Expect `or`
			errorType := p.expect(ast.TokenIdentifier).Value
			body = append(body, ast.EnsureNode{Condition: condition, ErrorType: errorType})
		} else if token.Type == ast.TokenReturn {
			p.advance() // Move past `return`
			returnExpression := p.parseExpression()
			/**
			 * TODO: Check if return expression is of the same type as the explicit return type
			 * This should probably be done in a separate type checking pass
			 */
			returnNode := ast.ReturnNode{Value: returnExpression, Type: returnExpression.ImplicitType()}
			body = append(body, returnNode)
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
func (p *Parser) parseFunctionDefinition() ast.FunctionNode {
	p.expect(ast.TokenFunction)           // Expect `fn`
	name := p.expect(ast.TokenIdentifier) // Function name

	params := p.parseFunctionSignature() // Parse function parameters

	explicitReturnType := p.parseReturnType()

	body := p.parseFunctionBody(explicitReturnType)

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
