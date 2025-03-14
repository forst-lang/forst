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

// Parse a function definition
func (p *Parser) parseFunctionDefinition() ast.FunctionNode {
	p.expect(ast.TokenFunction)           // Expect `fn`
	name := p.expect(ast.TokenIdentifier) // Function name

	params := p.parseFunctionSignature() // Parse function parameters
	var returnType ast.TypeNode
	if p.current().Type == ast.TokenColon {
		p.advance() // Consume the colon
		returnType = p.parseType()
	}

	p.expect(ast.TokenLBrace) // Expect `{`
	body := []ast.Node{}

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
			returnNode := ast.ReturnNode{Value: returnExpression, Type: returnType}
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

	return ast.FunctionNode{
		Name:       name.Value,
		ReturnType: returnType,
		Params:     params,
		Body:       body,
	}
}
