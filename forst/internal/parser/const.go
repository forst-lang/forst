package parser

import (
	"fmt"

	"forst/internal/ast"
)

func (p *Parser) parseConstDeclaration() ast.ConstGroupNode {
	p.advance() // consume 'const'
	if p.current().Type == ast.TokenLParen {
		p.advance()
		specs := p.parseConstSpecList()
		p.expect(ast.TokenRParen)
		return ast.ConstGroupNode{Specs: specs}
	}
	return ast.ConstGroupNode{Specs: []ast.ConstSpec{p.parseConstSpec(true)}}
}

func (p *Parser) parseConstSpecList() []ast.ConstSpec {
	var specs []ast.ConstSpec
	for p.current().Type != ast.TokenRParen {
		specs = append(specs, p.parseConstSpec(false))
	}
	return specs
}

func (p *Parser) parseConstSpec(requireValue bool) ast.ConstSpec {
	identTok := p.expect(ast.TokenIdentifier)
	spec := ast.ConstSpec{Name: ast.Ident{ID: ast.Identifier(identTok.Value)}}

	if p.current().Type == ast.TokenColon {
		p.advance()
		typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
		spec.Type = &typ
	}

	if p.current().Type == ast.TokenEquals {
		p.advance()
		spec.Value = p.parseExpression()
	} else if requireValue {
		p.FailWithParseError(identTok, fmt.Sprintf("const %s requires an initializer", identTok.Value))
	}

	return spec
}
