package parser

import (
	"forst/internal/ast"
)

func (p *Parser) parseSwitchStatement() ast.Node {
	p.advance() // switch

	if p.switchHeaderContainsTypeSwitch() {
		p.FailWithReport(p.current(), "switch-type-unsupported", "type switches are not supported",
			"type switches (switch v := x.(type)) are not supported.",
			"use narrowing instead: if x is Foo() { } else if x is Bar() { }")
	}

	var init ast.Node
	if p.current().Type == ast.TokenVar {
		p.advance()
		init = p.parseVarDeclaration()
		if p.current().Type != ast.TokenSemicolon {
			p.FailWithParseError(p.current(), "expected ';' after switch init statement")
		}
		p.advance()
	} else if p.current().Type == ast.TokenIdentifier && p.peek().Type == ast.TokenColonEquals {
		init = p.parseAssignment()
		if p.current().Type != ast.TokenSemicolon {
			p.FailWithParseError(p.current(), "expected ';' after switch init statement")
		}
		p.advance()
	}

	var tag ast.ExpressionNode
	if p.current().Type != ast.TokenLBrace {
		tag = p.parseExpression()
	}

	p.expect(ast.TokenLBrace)
	clauses := p.parseSwitchClauses()
	p.expect(ast.TokenRBrace)

	return &ast.SwitchNode{
		Init:    init,
		Tag:     tag,
		Clauses: clauses,
	}
}

func (p *Parser) switchHeaderContainsTypeSwitch() bool {
	for i := p.currentIndex; i < len(p.tokens); i++ {
		t := p.tokens[i]
		if t.Type == ast.TokenLBrace {
			break
		}
		if t.Value == "type" && i >= 2 &&
			p.tokens[i-1].Value == "(" && p.tokens[i-2].Value == "." {
			return true
		}
	}
	return false
}

func (p *Parser) parseSwitchClauses() []ast.SwitchClauseNode {
	var clauses []ast.SwitchClauseNode
	for p.current().Type != ast.TokenRBrace {
		switch p.current().Type {
		case ast.TokenCase:
			p.advance()
			values := p.parseSwitchCaseValues()
			p.expect(ast.TokenColon)
			body := p.parseSwitchClauseBody()
			clauses = append(clauses, ast.SwitchClauseNode{Values: values, Body: body})
		case ast.TokenDefault:
			p.advance()
			p.expect(ast.TokenColon)
			body := p.parseSwitchClauseBody()
			clauses = append(clauses, ast.SwitchClauseNode{IsDefault: true, Body: body})
		default:
			p.FailWithParseError(p.current(), "expected case or default in switch")
		}
	}
	return clauses
}

func (p *Parser) parseSwitchCaseValues() []ast.ExpressionNode {
	var values []ast.ExpressionNode
	for {
		values = append(values, p.parseExpression())
		if p.current().Type != ast.TokenComma {
			break
		}
		p.advance()
	}
	return values
}

func (p *Parser) parseSwitchClauseBody() []ast.Node {
	var body []ast.Node
	for {
		switch p.current().Type {
		case ast.TokenCase, ast.TokenDefault, ast.TokenRBrace:
			return body
		case ast.TokenFallthrough:
			p.advance()
			body = append(body, ast.FallthroughNode{})
		default:
			body = append(body, p.parseBlockStatement()...)
		}
	}
}
