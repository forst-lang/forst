package parser

import (
	"forst/pkg/ast"
)

func isPossibleConstraintIdentifier(token ast.Token) bool {
	return isCapitalCase(token.Value)
}

func (p *Parser) expectConstraintIdentifier() ast.Token {
	token := p.expect(ast.TokenIdentifier)
	if !isPossibleConstraintIdentifier(token) {
		panic(parseErrorMessage(token, "Constraint must start with capital letter"))
	}
	return token
}

func (p *Parser) parseConstraint() ast.ConstraintNode {
	// Each constraint must start with capital letter
	constraint := p.expectConstraintIdentifier()

	var args []ast.ValueNode

	token := p.current()
	// Constraints must have parentheses
	if token.Type != ast.TokenLParen {
		panic(parseErrorMessage(constraint, "Constraint must have parentheses"))
	}

	p.advance() // Consume (

	// Parse arguments until closing parenthesis
	for p.current().Type != ast.TokenRParen {
		value := p.parseValue()
		args = append(args, value)

		if p.current().Type == ast.TokenComma {
			p.advance()
		}
	}
	p.expect(ast.TokenRParen)

	return ast.ConstraintNode{
		Name: constraint.Value,
		Args: args,
	}
}

func (p *Parser) parseAssertionChain(requireBaseType bool) ast.AssertionNode {
	var constraints []ast.ConstraintNode
	var baseType *ast.TypeIdent

	// Parse optional base type (must start with capital letter)
	token := p.current()
	if isPossibleTypeIdentifier(token) || isPossibleConstraintIdentifier(token) {
		// If next token is not a parenthesis, this is a base type
		if isPossibleTypeIdentifier(token) && p.peek().Type != ast.TokenLParen {
			typ := p.parseType()
			baseType = &typ.Name
		} else {
			if requireBaseType {
				panic(parseErrorMessage(token, "Expected base type for assertion"))
			}
			// Otherwise it's a constraint
			constraint := p.parseConstraint()
			constraints = append(constraints, constraint)
		}
	}

	// Parse chain of assertions
	for p.current().Type == ast.TokenDot {
		p.advance() // Consume dot
		constraint := p.parseConstraint()
		constraints = append(constraints, constraint)
	}

	return ast.AssertionNode{
		BaseType:    baseType,
		Constraints: constraints,
	}
}
