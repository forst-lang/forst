package parser

import (
	"forst/pkg/ast"
	"unicode"
)

func (p *Parser) expectTypeIdentifier() ast.Token {
	token := p.expect(ast.TokenType)
	if !unicode.IsUpper(rune(token.Value[0])) {
		panic(parseErrorWithValue(token, "Expected type identifier to start with an uppercase letter"))
	}
	return token
}

func (p *Parser) parseTypeDef() ast.TypeDefNode {
	name := p.expectTypeIdentifier()
	typeIdent := ast.TypeIdent(name.Value)

	p.expect(ast.TokenIs)
	p.advance()

	expr := p.parseTypeDefExpr()

	return ast.TypeDefNode{
		Ident: typeIdent,
		Expr:  expr,
	}
}

func (p *Parser) parseTypeDefExpr() ast.TypeDefExpr {
	if p.current().Type == ast.TokenLParen {
		p.advance()
		expr := p.parseTypeDefExpr()
		p.expect(ast.TokenRParen)
		p.advance()
		return expr
	}

	var left ast.TypeDefExpr
	token := p.expectTypeIdentifier()
	if p.peek().Type == ast.TokenDot {
		assertion := p.parseAssertionChain()
		left = ast.TypeDefAssertionExpr{
			Assertion: &assertion,
		}
	} else {
		typeIdent := ast.TypeIdent(token.Value)
		return &ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &typeIdent,
			},
		}
	}

	if p.current().Type != ast.TokenBitwiseAnd && p.current().Type != ast.TokenBitwiseOr {
		return left
	}

	op := p.current().Type
	p.advance()

	right := p.parseTypeDefExpr()

	return ast.TypeDefBinaryExpr{
		Left:  left,
		Op:    op,
		Right: right,
	}
}
