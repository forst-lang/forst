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

func (p *Parser) parseEnsureStatement(context *Context) ast.EnsureNode {
	p.advance() // Move past `ensure`

	variable := p.expect(ast.TokenIdentifier).Value

	p.expect(ast.TokenIs)

	condition := p.parseAssertionChain(context)

	if !context.IsMainFunction() || p.current().Type == ast.TokenOr {
		p.expect(ast.TokenOr) // Expect `or`
		errorType := p.expect(ast.TokenIdentifier).Value
		var err ast.EnsureErrorNode
		if p.current().Type == ast.TokenLParen {
			p.advance() // Consume left paren
			var args []ast.ExpressionNode
			for p.current().Type != ast.TokenRParen {
				args = append(args, p.parseExpression(context))
				if p.current().Type == ast.TokenComma {
					p.advance()
				}
			}
			p.expect(ast.TokenRParen)
			err = ast.EnsureErrorCall{ErrorType: errorType, ErrorArgs: args}
		} else {
			err = ast.EnsureErrorVar(errorType)
		}
		return ast.EnsureNode{Variable: variable, Assertion: condition, Error: &err}
	}

	return ast.EnsureNode{Variable: variable, Assertion: condition}
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
			body = append(body, p.parseReturnStatement(context))
		} else if token.Type == ast.TokenIdentifier {
			// Look ahead to see if this is a function call or assignment
			if p.peek().Type == ast.TokenLParen {
				// Function call
				body = append(body, p.parseExpression(context))
			} else if p.peek().Type == ast.TokenComma {
				// Multiple assignment
				firstIdent := p.expect(ast.TokenIdentifier)
				p.expect(ast.TokenComma)
				secondIdent := p.expect(ast.TokenIdentifier)

				// Expect assignment operator
				assignToken := p.current()
				if assignToken.Type != ast.TokenEquals && assignToken.Type != ast.TokenColonEquals {
					panic(parseErrorWithValue(assignToken, "Expected assignment or short assignment operator"))
				}
				p.advance()

				// Parse comma-separated expressions
				var exprs []ast.ExpressionNode
				for {
					exprs = append(exprs, p.parseExpression(context))
					if p.current().Type != ast.TokenComma {
						break
					}
					p.advance() // Skip comma
				}

				body = append(body, ast.AssignmentNode{
					Names:         []string{firstIdent.Value, secondIdent.Value},
					Values:        exprs,
					ExplicitTypes: []*ast.TypeNode{nil, nil},
					IsShort:       assignToken.Type == ast.TokenColonEquals,
				})
			} else {
				panic(parseErrorWithValue(token, "Expected function call or assignment after identifier"))
			}
		} else {
			panic(parseErrorWithValue(token, "Unexpected token in function body"))
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
