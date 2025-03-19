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

func (p *Parser) parseConstraintArgument() ast.ConstraintArgumentNode {
	// Check if this is a shape literal
	if p.current().Type == ast.TokenLBrace {
		shape := p.parseShape()
		return ast.ConstraintArgumentNode{
			Shape: &shape,
		}
	}

	// Otherwise parse as a regular value or type assertion
	if p.current().Type == ast.TokenLBrace {
		shape := p.parseShape()
		return ast.ConstraintArgumentNode{
			Shape: &shape,
		}
	}
	value := p.parseValue()
	return ast.ConstraintArgumentNode{
		Value: &value,
	}
}

func (p *Parser) parseShape() ast.ShapeNode {
	p.expect(ast.TokenLBrace)

	fields := make(map[string]ast.ShapeFieldNode)
	// Parse fields until closing brace
	for p.current().Type != ast.TokenRBrace {
		// Parse field name
		name := p.expect(ast.TokenIdentifier).Value
		p.expect(ast.TokenColon)

		// Parse field value (can be another shape or a type assertion)
		var value ast.ShapeFieldNode
		if p.current().Type == ast.TokenLBrace {
			shape := p.parseShape()
			value = ast.ShapeFieldNode{
				Shape: &shape,
			}
		} else {
			assertion := p.parseAssertionChain(true)
			value = ast.ShapeFieldNode{
				Assertion: &assertion,
			}
		}

		fields[name] = value

		// Handle commas between fields
		if p.current().Type != ast.TokenRBrace {
			p.expect(ast.TokenComma)
		}
	}

	p.expect(ast.TokenRBrace)

	return ast.ShapeNode{Fields: fields}
}

func (p *Parser) parseConstraint() ast.ConstraintNode {
	constraint := p.expectConstraintIdentifier()
	p.expect(ast.TokenLParen)

	var args []ast.ConstraintArgumentNode
	for p.current().Type != ast.TokenRParen {
		arg := p.parseConstraintArgument()
		args = append(args, arg)

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
