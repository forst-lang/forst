package parser

import (
	"forst/internal/ast"
)

// parseTypeGuard parses a type guard declaration
func (p *Parser) parseTypeGuard() *ast.TypeGuardNode {
	p.expect(ast.TokenIs)
	p.expect(ast.TokenLParen)

	// Parse subject parameter (required)
	if p.current().Type == ast.TokenRParen {
		panic(parseErrorMessage(p.current(), "type guard requires a subject parameter"))
	}
	subjectParam := p.parseParameter()

	// Parse additional parameters if any
	var additionalParams []ast.ParamNode
	for p.current().Type != ast.TokenRParen {
		if p.current().Type == ast.TokenComma {
			p.advance()
			if p.current().Type == ast.TokenRParen {
				panic(parseErrorMessage(p.current(), "trailing comma in type guard parameters"))
			}
		}
		param := p.parseParameter()
		additionalParams = append(additionalParams, param)
	}
	p.expect(ast.TokenRParen)

	// Parse type guard name
	name := p.expect(ast.TokenIdentifier)

	// Parse body
	body := p.parseBlock(&BlockContext{AllowReturn: true})

	return &ast.TypeGuardNode{
		Ident:            ast.Identifier(name.Value),
		SubjectParam:     subjectParam,
		AdditionalParams: additionalParams,
		Body:             body,
	}
}
