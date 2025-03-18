package parser

import (
	"fmt"
	"unicode"

	"forst/pkg/ast"
)

func (p *Parser) parseConstraint() ast.ConstraintNode {
	// Each assertion must start with capital letter
	assertion := p.expect(ast.TokenIdentifier)
	if !unicode.IsUpper(rune(assertion.Value[0])) {
		panic(fmt.Sprintf("Assertion must start with capital letter: %s", assertion.Value))
	}

	var args []ast.ValueNode

	// Constraints must have parentheses
	if p.current().Type != ast.TokenLParen {
		panic(fmt.Sprintf("Constraint must have parentheses: %s", assertion.Value))
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
		Name: assertion.Value,
		Args: args,
	}
}

func (p *Parser) parseAssertionChain() ast.AssertionNode {
	var constraints []ast.ConstraintNode
	var baseType *ast.TypeIdent

	// Parse optional base type (must start with capital letter)
	if p.current().Type == ast.TokenIdentifier && unicode.IsUpper(rune(p.current().Value[0])) {
		// If next token is not a parenthesis, this is a base type
		if p.peek().Type != ast.TokenLParen {
			typeIdent := ast.TypeIdent(p.current().Value)
			baseType = &typeIdent
			p.advance()
		} else {
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
