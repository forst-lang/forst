package parser

import (
	"forst/pkg/ast"
)

func (p *Parser) expectCustomTypeIdentifier() ast.Token {
	token := p.expect(ast.TokenIdentifier)
	if !isPossibleTypeIdentifier(token) {
		panic(parseErrorWithValue(token, "Expected type identifier to start with an uppercase letter"))
	}
	return token
}

func (p *Parser) expectCustomTypeIdentifierOrPackageName() ast.Token {
	if p.peek().Type == ast.TokenDot && isPossibleTypeIdentifier(p.peek(2)) && p.peek(3).Type != ast.TokenLParen {
		return p.expect(ast.TokenIdentifier)
	}
	return p.expectCustomTypeIdentifier()
}

func (p *Parser) parseTypeDef() *ast.TypeDefNode {
	p.expect(ast.TokenType)

	name := p.expectCustomTypeIdentifier()
	typeIdent := ast.TypeIdent(name.Value)

	p.expect(ast.TokenEquals)

	expr := p.parseTypeDefExpr()

	return &ast.TypeDefNode{
		Ident: typeIdent,
		Expr:  expr,
	}
}

func (p *Parser) parseTypeDefExpr() ast.TypeDefExpr {
	if p.current().Type == ast.TokenLParen {
		p.advance()
		expr := p.parseTypeDefExpr()
		p.expect(ast.TokenRParen)
		return expr
	}

	var left ast.TypeDefExpr
	if p.peek().Type == ast.TokenDot {
		assertion := p.parseAssertionChain(true)
		left = ast.TypeDefAssertionExpr{
			Assertion: &assertion,
		}
	} else {
		typ := p.parseType()

		return &ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &typ.Name,
			},
		}
	}

	operator := p.current()
	if operator.Type != ast.TokenBitwiseAnd && operator.Type != ast.TokenBitwiseOr {
		return left
	}

	p.advance()

	right := p.parseTypeDefExpr()

	return ast.TypeDefBinaryExpr{
		Left:  left,
		Op:    operator.Type,
		Right: right,
	}
}
