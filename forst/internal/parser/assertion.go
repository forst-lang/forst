package parser

import (
	"forst/internal/ast"
)

func isPossibleConstraintIdentifier(token ast.Token) bool {
	return isCapitalCase(token.Value)
}

func (p *Parser) expectConstraintIdentifier() ast.Token {
	token := p.expect(ast.TokenIdentifier)
	if !isPossibleConstraintIdentifier(token) {
		p.log.Fatalf("%s", parseErrorMessage(token, "Constraint must start with capital letter"))
	}
	return token
}

func (p *Parser) parseConstraintArgument() ast.ConstraintArgumentNode {
	token := p.current()
	// Allow shape types as type arguments
	if token.Type == ast.TokenLBrace {
		typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
		return ast.ConstraintArgumentNode{
			Type: &typ,
		}
	}

	// If this is a type (identifier, shape, etc), parse as TypeNode
	if isPossibleTypeIdentifier(token, TypeIdentOpts{AllowLowercaseTypes: true}) ||
		token.Type == ast.TokenLBracket ||
		token.Type == ast.TokenMap ||
		token.Type == ast.TokenStar {
		typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
		return ast.ConstraintArgumentNode{
			Type: &typ,
		}
	}

	// Otherwise, parse as value
	value := p.parseValue()
	return ast.ConstraintArgumentNode{
		Value: &value,
	}
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

	token := p.current()
	isIdentOrConstraint := token.Type == ast.TokenIdentifier || isPossibleConstraintIdentifier(token)

	if isIdentOrConstraint {
		isConstraintWithoutBaseType := p.peek().Type == ast.TokenLParen

		if isConstraintWithoutBaseType {
			if requireBaseType {
				p.FailWithParseError(token, "Expected base type for assertion")
			}
			constraint := p.parseConstraint()
			constraints = append(constraints, constraint)
		} else {
			// Parse first segment (could be package name or type)
			typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
			baseType = &typ.Ident

			// Check if it's a package name
			if p.current().Type == ast.TokenDot {
				nextToken := p.peek()
				isQualifiedType := isPossibleTypeIdentifier(nextToken, TypeIdentOpts{AllowLowercaseTypes: false}) &&
					p.peek(2).Type != ast.TokenLParen

				if isQualifiedType {
					p.advance() // Consume dot
					pkgType := p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
					qualifiedName := ast.TypeIdent(string(*baseType) + "." + string(pkgType.Ident))
					baseType = &qualifiedName
				}
			}
		}
	}

	// Parse constraint chain
	for p.current().Type == ast.TokenDot {
		p.advance()
		constraint := p.parseConstraint()
		constraints = append(constraints, constraint)
	}

	return ast.AssertionNode{
		BaseType:    baseType,
		Constraints: constraints,
	}
}
